package config

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client
var MongoDB *mongo.Database
var MongoChatCollection *mongo.Collection

func InitMongoDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoURI := "mongodb://localhost:27017"
	dbName := "db_chat_ta"

	clientOpts := options.Client().ApplyURI(mongoURI)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("gagal terhubung ke MongoDB: %w", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return fmt.Errorf("gagal melakukan ping ke MongoDB: %w", err)
	}

	log.Println("✅ Terhubung ke basis data MongoDB!")

	MongoClient = client
	MongoDB = client.Database(dbName)
	MongoChatCollection = MongoDB.Collection("conversations")

	return nil
}
