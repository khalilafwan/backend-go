package controllers

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"backend-go/config"
	"backend-go/models"
	"backend-go/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func UploadVoiceHandler(c *gin.Context) {
	// 1. Ambil form field chat_id dan user_id
	chatID := c.PostForm("chat_id")
	log.Println("Menerima chatID dari form:", chatID)
	userID := c.MustGet("userID").(int)

	// 2. Ambil file audio
	file, err := c.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gagal mendapatkan berkas audio"})
		return
	}

	// Tambahkan delay kecil agar write selesai (opsional)
	time.Sleep(500 * time.Millisecond)
	// 3. Simpan file sementara ke disk
	inputPath := "/tmp/" + file.Filename
	if err := c.SaveUploadedFile(file, inputPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan berkas audio"})
		return
	}

	// (Opsional) cek ukuran file
	fileInfo, err := os.Stat(inputPath)
	if err != nil || fileInfo.Size() < 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File audio terlalu kecil atau corrupt"})
		return
	}

	// 4. Konversi dari webm ke mp3
	outputPath := "/tmp/converted_" + strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename)) + ".mp3"
	outFile, err := os.Create(outputPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat file output"})
		return
	}
	defer outFile.Close()

	info, err := os.Stat(inputPath)
	if err != nil {
		log.Printf("❌ Gagal menemukan file input: %v", err)
		return
	}
	log.Printf("📁 Ukuran file input: %d bytes", info.Size())

	cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, outputPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("⚠️ Peringatan atau kesalahan FFmpeg: %s", output)
		// Jangan langsung return 500, cek dulu apakah file hasil ada & valid
	}

	log.Printf("Menjalankan perintah ffmpeg: ffmpeg -i %s %s", inputPath, outputPath)

	// Cek apakah file hasil ada dan tidak kosong
	fi, statErr := os.Stat(outputPath)
	if statErr != nil || fi.Size() == 0 {
		log.Printf("❌ Berkas output tidak ditemukan atau kosong: %v", statErr)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Konversi MP3 gagal atau menghasilkan file kosong.",
		})
		return
	}

	// Kalau sampai sini, konversi dianggap berhasil meskipun ada err
	log.Printf("✅ Konversi MP3 berhasil, ukuran: %d bytes", fi.Size())

	// 5. Baca hasil MP3 sebagai []byte
	mp3Bytes, err := os.ReadFile(outputPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca file MP3"})
		return
	}

	// Debug step — success log
	log.Println("Konversi MP3 berhasil, ukuran:", len(mp3Bytes))

	// Log return sebagai response base64 atau hanya info berhasil
	log.Printf("Konversi MP3 siap untuk transkripsi, chatID: %s, userID: %d", chatID, userID)

	// 6. Kirim ke AWS Transcribe
	transcript, s3Uri, err := services.TranscribeAudio(mp3Bytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mentranskrip audio"})
		return
	}

	// 7. Kirim transcript ke NLP Flask
	nlpResp, err := services.CallNLPService(transcript)
	log.Println("🎯 Respons dari NLP:", nlpResp.ResponseMessage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendapatkan respons NLP"})
		return
	}

	// 8. Kirim intent ke TTS
	botVoiceBytes, err := services.SynthesizeSpeech(nlpResp.ResponseMessage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengonversi teks menjadi suara"})
		return
	}

	// 9. Upload hasil bot voice ke S3
	botAudioURL, err := services.UploadBotVoiceToS3("bot_"+file.Filename, botVoiceBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengunggah audio bot ke S3"})
		return
	}

	// 10. Simpan ke MongoDB voice_messages
	err = services.SaveVoiceChatHistory(chatID, userID, transcript, nlpResp.Intent, s3Uri, botAudioURL, nlpResp.ResponseMessage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pesan suara"})
		return
	}

	// 11. Bersihkan file sementara
	os.Remove(inputPath)
	os.Remove(outputPath)

	// 12. Balas ke frontend
	c.JSON(http.StatusOK, gin.H{
		"transcript":    transcript,
		"intent":        nlpResp.Intent,
		"bot_audio_url": botAudioURL,
	})
}

func GetVoiceMessages(chatID string) ([]models.VoiceMessage, error) {
	var messages []models.VoiceMessage
	collection := config.MongoDB.Collection("voice_messages")

	filter := bson.M{"chat_id": chatID}
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})
	cursor, err := collection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var msg models.VoiceMessage
		if err := cursor.Decode(&msg); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func GetVoiceMessagesByID(c *gin.Context) {
	chatID := c.Param("chatID")
	ctx := context.TODO()

	filter := bson.M{"chat_id": chatID}
	cursor, err := config.MongoDB.Collection("voice_messages").Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil voice messages"})
		return
	}
	defer cursor.Close(ctx)

	var messages []models.VoiceMessage
	if err := cursor.All(ctx, &messages); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses data voice messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}

func ServeAudioFile(c *gin.Context) {
	filename := c.Param("filename")
	filePath := filepath.Join("uploads", filename)

	// Buka file audio
	file, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File audio tidak ditemukan"})
		return
	}
	defer file.Close()

	// Set header konten audio
	c.Header("Content-Type", "audio/mpeg") // sesuaikan tipe jika format lain
	c.Header("Content-Disposition", "inline; filename="+filename)
	c.Status(http.StatusOK)
	io.Copy(c.Writer, file)
}
