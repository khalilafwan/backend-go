package services

import (
	"fmt"

	"backend-go/config"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// struct untuk response frontend
type ConversationSummary struct {
	ChatID        string `bson:"chat_id" json:"chat_id"`
	Username      string `bson:"username" json:"username"`
	LastMessageAt string `bson:"last_message_at" json:"last_message_at"`
}

// GetAdminMetrics:
// - totalUsers -> dari PostgreSQL (tabel users)
// - totalConvos -> dari MongoDB (jumlah dokumen pada collection conversations)
// - totalMsgs -> dari MongoDB (jumlah total elemen pesan di field messages pada tiap dokumen)
func GetAdminMetrics(c *gin.Context) (map[string]int64, error) {
	var totalUsers int64

	// 1) Ambil total users dari Postgres
	queryUsers := `SELECT COUNT(*) FROM "users"`
	err := config.DB.QueryRow(queryUsers).Scan(&totalUsers)
	if err != nil {
		config.Log.Error("Error retrieving total users: ", err)
		return nil, err
	}

	// 2) Ambil total conversations (count documents) dari MongoDB
	convCollection := config.MongoDB.Collection("conversations")
	ctx := c.Request.Context()

	totalConvos, err := convCollection.CountDocuments(ctx, bson.D{})
	if err != nil {
		config.Log.Error("Error counting conversations in MongoDB: ", err)
		return nil, err
	}

	// 3) Hitung total messages menggunakan aggregation: sum ukuran array messages di setiap dokumen
	// pipeline:
	//  - project messagesCount = size(ifNull(messages, []))
	//  - group _id: null, total: sum(messagesCount)
	pipeline := mongoPipelineForMessageCount()
	cursor, err := convCollection.Aggregate(ctx, pipeline)
	if err != nil {
		config.Log.Error("Error aggregating total messages in MongoDB: ", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var totalMsgs int64 = 0
	if cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			config.Log.Error("Error decoding aggregation result: ", err)
			return nil, err
		}
		// doc["total"] bisa berupa int32/int64/float64 tergantung hasil driver
		if v, ok := doc["total"]; ok && v != nil {
			switch n := v.(type) {
			case int32:
				totalMsgs = int64(n)
			case int64:
				totalMsgs = n
			case float64:
				totalMsgs = int64(n)
			default:
				// fallback: fmt.Sprint -> parse if needed
				var parsed int64
				_, _ = fmt.Sscan(fmt.Sprint(n), &parsed)
				totalMsgs = parsed
			}
		}
	}
	// check cursor error
	if err := cursor.Err(); err != nil {
		config.Log.Error("Cursor error during messages aggregation: ", err)
		return nil, err
	}

	metrics := map[string]int64{
		"total_users":         totalUsers,
		"total_conversations": totalConvos,
		"total_messages":      totalMsgs,
	}

	return metrics, nil
}

// helper: buat pipeline untuk menghitung total messages
func mongoPipelineForMessageCount() mongoPipeline {
	// return pipeline as []bson.M or []bson.D depending on style
	// We define as []bson.M for readability
	return mongoPipeline{
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "messagesCount", Value: bson.D{
				{Key: "$size", Value: bson.D{
					{Key: "$ifNull", Value: bson.A{"$messages", bson.A{}}},
				}},
			}},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$messagesCount"}}},
		}}},
	}
}

// typedef-like alias to keep pipeline helper tidy
type mongoPipeline = []bson.D

// GetRecentConversations: ambil dokumen conversations terbaru dari MongoDB
// Kembalikan slice ConversationSummary dengan batas limit 10 (atau parameter dari query jika ingin)
func GetRecentConversations(c *gin.Context) ([]ConversationSummary, error) {
	var convos []ConversationSummary

	convCollection := config.MongoDB.Collection("conversations")
	ctx := c.Request.Context()

	// limit optional via query param ?limit=
	limit := int64(10)
	if l := c.Query("limit"); l != "" {
		var tmp int
		if _, err := fmt.Sscan(l, &tmp); err == nil && tmp > 0 {
			limit = int64(tmp)
		}
	}

	opts := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}}).SetLimit(limit)

	cursor, err := convCollection.Find(ctx, bson.D{}, opts)
	if err != nil {
		config.Log.Error("Error retrieving recent conversations from MongoDB: ", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			config.Log.Error("Error decoding conversation document: ", err)
			return nil, err
		}

		// Ambil fields: chat_id, username, last_message_at
		cs := ConversationSummary{
			ChatID:        toString(doc["chat_id"]),
			Username:      toString(doc["username"]),
			LastMessageAt: "",
		}

		// Preferensi: ambil timestamp dari field updated_at jika ada,
		// atau ambil waktu pesan terakhir dari messages array (messages[-1].created_at)
		if ua, ok := doc["updated_at"]; ok && ua != nil {
			cs.LastMessageAt = fmt.Sprint(ua)
		} else if msgs, ok := doc["messages"]; ok && msgs != nil {
			// messages mungkin array of bson.M
			if arr, ok := msgs.(bson.A); ok && len(arr) > 0 {
				last := arr[len(arr)-1]
				if m, ok := last.(bson.M); ok {
					if t, ok := m["created_at"]; ok && t != nil {
						cs.LastMessageAt = fmt.Sprint(t)
					} else if t2, ok := m["timestamp"]; ok && t2 != nil {
						cs.LastMessageAt = fmt.Sprint(t2)
					} else {
						// fallback: serialize the message or set empty
						cs.LastMessageAt = ""
					}
				}
			}
		}

		convos = append(convos, cs)
	}

	if err := cursor.Err(); err != nil {
		config.Log.Error("Cursor error after iterating recent conversations: ", err)
		return nil, err
	}

	return convos, nil
}

// toString: helper kecil untuk mengkonversi interface{} ke string aman
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		return fmt.Sprint(x)
	}
}
