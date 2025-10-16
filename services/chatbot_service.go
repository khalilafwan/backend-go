package services

import (
	"backend-go/config"
	"backend-go/models"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ChatbotService struct {
	ChatCollection *mongo.Collection
}

type ChatListItem struct {
	ChatID    string    `json:"chat_id"`
	ChatTitle string    `json:"chat_title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ChatMessageResponse struct {
	Sender     string    `json:"sender"`               // "user" atau "bot"
	Type       string    `json:"type"`                 // "text" atau "voice"
	Message    string    `json:"message,omitempty"`    // Hanya untuk text
	AudioURL   string    `json:"audio_url,omitempty"`  // Hanya untuk voice
	Transcript string    `json:"transcript,omitempty"` // Hanya untuk voice
	Intent     string    `json:"intent,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// Fungsi utama untuk memproses pesan user
func ProcessChatbot(chatID string, userMessage string, userID int, username string) (*models.ChatbotResponse, error) {
	startTime := time.Now()

	// Panggil layanan NLP (Flask)
	nlpResp, err := CallNLPService(userMessage)
	if err != nil {
		return nil, err
	}

	userMsg := models.Message{
		Sender:    "user",
		Message:   userMessage,
		Timestamp: startTime.Format(time.RFC3339),
	}

	botMsg := models.Message{
		Sender:     "bot",
		Message:    nlpResp.ResponseMessage,
		Intent:     nlpResp.Intent,
		Confidence: nlpResp.Confidence,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
	// Simpan ke MongoDB
	err = SaveToMongo(chatID, userID, username, userMsg, botMsg)
	if err != nil {
		return nil, fmt.Errorf("gagal menyimpan ke MongoDB: %v", err)
	}

	// Update last_chat_id ke PostgreSQL
	if err := UpdateLastChatID(userID, chatID); err != nil {
		fmt.Println("Gagal update last_chat_id:", err)
		// kamu bisa log tapi tidak harus menghentikan proses jika error
	}

	// Buat respons ke frontend
	shouldEscalate := nlpResp.Confidence < 0.6

	return &models.ChatbotResponse{
		ChatID:   chatID,
		Intent:   nlpResp.Intent,
		Message:  nlpResp.ResponseMessage,
		Escalate: shouldEscalate,
	}, nil
}

func SaveToMongo(chatID string, userID int, username string, userMsg, botMsg models.Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := config.MongoDB.Collection("conversations")

	filter := bson.M{"chat_id": chatID}
	var existing bson.M
	err := collection.FindOne(ctx, filter).Decode(&existing)

	now := time.Now()

	if err != nil {
		// Dokumen baru
		convo := models.Conversation{
			ChatID:    chatID,
			UserID:    userID,
			Username:  username,
			ChatTitle: fmt.Sprintf("Percakapan pada %s", now.Format("2 January 2006 15:04")),
			Messages:  []models.Message{userMsg, botMsg},
			CreatedAt: now,
			UpdatedAt: now,
		}

		_, insertErr := collection.InsertOne(ctx, convo)
		return insertErr
	}

	// Jika sudah ada, push message baru dan update waktu
	update := bson.M{
		"$push": bson.M{
			"messages": bson.M{
				"$each": []models.Message{userMsg, botMsg},
			},
		},
		"$set": bson.M{
			"updated_at": now,
		},
	}

	_, updateErr := collection.UpdateOne(ctx, filter, update)
	return updateErr
}

func UpdateLastChatID(userID int, chatID string) error {
	db := config.DB

	query := `UPDATE users SET last_chat_id = $1 WHERE id = $2`
	_, err := db.Exec(query, chatID, userID)
	if err != nil {
		return fmt.Errorf("gagal update last_chat_id di PostgreSQL: %v", err)
	}
	return nil
}

func GetChatByID(chatID string, userID int) (*models.Conversation, error) {
	var convo models.Conversation
	err := config.MongoDB.Collection("conversations").FindOne(
		context.TODO(),
		bson.M{"chat_id": chatID, "user_id": userID},
	).Decode(&convo)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil // Chat tidak ditemukan
		}
		return nil, err // Error lainnya (koneksi, dsb)
	}

	return &convo, nil
}

func FetchUserChatList(userID int) ([]ChatListItem, error) {
	collection := config.MongoDB.Collection("conversations")

	filter := bson.M{"user_id": userID}
	opts := options.Find().SetSort(bson.M{"updated_at": -1}) // urut terbaru duluan

	cursor, err := collection.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var results []ChatListItem
	for cursor.Next(context.Background()) {
		var conv models.Conversation
		if err := cursor.Decode(&conv); err != nil {
			continue
		}

		results = append(results, ChatListItem{
			ChatID:    conv.ChatID,
			ChatTitle: conv.ChatTitle,
			UpdatedAt: conv.UpdatedAt,
		})
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func RenameChatTitle(chatID string, userID int, newTitle string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := config.MongoDB.Collection("conversations")

	filter := bson.M{"chat_id": chatID, "user_id": userID}
	update := bson.M{"$set": bson.M{"chat_title": newTitle}}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("chat tidak ditemukan atau tidak diizinkan")
	}

	return nil
}

func DeleteChat(chatID string, userID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := config.MongoDB.Collection("conversations")
	filter := bson.M{"chat_id": chatID, "user_id": userID}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("chat tidak ditemukan atau tidak diizinkan")
	}

	return nil
}

func GetFullChatHistory(chatID string, userID int) ([]ChatMessageResponse, error) {
	ctx := context.TODO()

	// 1. Ambil data Conversation (text)
	var convo models.Conversation
	err := config.MongoDB.Collection("conversations").FindOne(ctx, bson.M{
		"chat_id": chatID,
		"user_id": userID,
	}).Decode(&convo)

	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, err
	}

	// 2. Ambil data VoiceMessage (suara)
	cursor, err := config.MongoDB.Collection("voice_messages").Find(ctx, bson.M{
		"chat_id": chatID,
		"user_id": userID,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var voiceMessages []models.VoiceMessage
	if err := cursor.All(ctx, &voiceMessages); err != nil {
		return nil, err
	}

	// 3. Gabungkan dua tipe message
	var combined []ChatMessageResponse

	// Text messages
	for _, msg := range convo.Messages {
		t, _ := time.Parse(time.RFC3339, msg.Timestamp)
		combined = append(combined, ChatMessageResponse{
			Sender:    msg.Sender,
			Type:      "text",
			Message:   msg.Message,
			Intent:    msg.Intent,
			Timestamp: t,
		})
	}

	// Voice messages
	for _, voice := range voiceMessages {
		combined = append(combined, ChatMessageResponse{
			Sender:     voice.Sender,
			Type:       "voice",
			AudioURL:   voice.AudioURL,
			Transcript: voice.Transcript,
			Intent:     voice.Intent,
			Timestamp:  voice.Timestamp,
		})
	}

	// 4. Urutkan berdasarkan timestamp
	sort.Slice(combined, func(i, j int) bool {
		return combined[i].Timestamp.Before(combined[j].Timestamp)
	})

	return combined, nil
}

// ==== Fungsi untuk panggil NLP Flask ====

type nlpResponse struct {
	Intent          string  `json:"intent"`
	ResponseMessage string  `json:"response_message"`
	Confidence      float64 `json:"confidence"`
}

func CallNLPService(message string) (*nlpResponse, error) {
	requestBody, err := json.Marshal(map[string]string{
		"message": message,
	})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post("http://localhost:5000/nlp", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("gagal memanggil layanan NLP: %v", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var nlpResp nlpResponse
	if err := json.Unmarshal(body, &nlpResp); err != nil {
		return nil, fmt.Errorf("gagal memproses respons NLP: %v", err)
	}

	return &nlpResp, nil
}
