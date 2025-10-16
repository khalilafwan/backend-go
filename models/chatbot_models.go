package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ==== Bagian: Chatbot Request & Response ====

type ChatbotRequest struct {
	Message string `json:"message" binding:"required"`
}

type ChatbotResponse struct {
	ChatID   string `json:"chat_id"`
	Intent   string `json:"intent"`
	Message  string `json:"message"`
	Escalate bool   `json:"escalate"`
}

// ==== Bagian: MongoDB Conversation ====

type Message struct {
	Sender     string  `bson:"sender" json:"sender"`
	Message    string  `bson:"message" json:"message"`
	Intent     string  `bson:"intent,omitempty" json:"intent,omitempty"`
	Confidence float64 `bson:"confidence,omitempty" json:"confidence,omitempty"`
	Timestamp  string  `bson:"timestamp" json:"timestamp"`
}

type Conversation struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChatID    string             `bson:"chat_id" json:"chat_id"`
	UserID    int                `bson:"user_id" json:"user_id"`
	Username  string             `bson:"username" json:"username"`
	ChatTitle string             `bson:"chat_title" json:"chat_title"`
	Messages  []Message          `bson:"messages" json:"messages"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

type IntentSummary struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID   int                `bson:"user_id" json:"user_id"`
	Intent   string             `bson:"intent" json:"intent"`
	Count    int                `bson:"count" json:"count"`
	LastUsed time.Time          `bson:"last_used" json:"last_used"`
}

type VoiceMessage struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	ChatID     string             `bson:"chat_id"`
	UserID     int                `bson:"user_id" json:"user_id"`
	Sender     string             `bson:"sender"`
	AudioURL   string             `bson:"audio_url"`
	Transcript string             `bson:"transcript"`
	Intent     string             `bson:"intent"`
	Timestamp  time.Time          `bson:"timestamp"`
}

type ChatMessageResponse struct {
	Type       string    `json:"type"`
	Sender     string    `json:"sender"`
	Content    string    `json:"content,omitempty"`
	AudioURL   string    `json:"audio_url,omitempty"`
	Transcript string    `json:"transcript,omitempty"`
	Intent     string    `json:"intent,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}
