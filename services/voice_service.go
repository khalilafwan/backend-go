package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend-go/config"
	"backend-go/models"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/transcribe"
	transcribeTypes "github.com/aws/aws-sdk-go-v2/service/transcribe/types"
)

var (
	transcribeClient *transcribe.Client
	s3Client         *s3.Client
	httpClient       = http.Client{Timeout: 10 * time.Second} // HTTP client untuk ambil hasil transkripsi
)

// InitVoiceServices inisialisasi klien Polly, Transcribe, dan S3
func InitVoiceServices() {
	cfg := config.AWSConfig
	transcribeClient = transcribe.NewFromConfig(cfg)
	s3Client = s3.NewFromConfig(cfg)
	log.Println("✅ Layanan suara (Transcribe & S3) telah diinisialisasi.")
}

// SynthesizeSpeech mengubah teks menjadi audio mp3 menggunakan Amazon Polly
func SynthesizeSpeech(text string) ([]byte, error) {
	log.Println("📝 Teks yang diterima oleh Google TTS:", fmt.Sprintf("[%s]", text))

	ctx := context.Background()

	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		log.Println("❌ Gagal membuat client:", err)
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	req := &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: text},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: "id-ID",
			SsmlGender:   texttospeechpb.SsmlVoiceGender_MALE,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_MP3,
		},
	}

	resp, err := client.SynthesizeSpeech(ctx, req)
	if err != nil {
		log.Println("❌ Gagal synthesize:", err)
		return nil, fmt.Errorf("failed to synthesize: %w", err)
	}

	if len(resp.AudioContent) == 0 {
		log.Println("⚠️ TTS menghasilkan audio kosong.")
		return nil, fmt.Errorf("empty audio content")
	}

	log.Printf("✅ Suara bot sudah disintesis, size: %d bytes", len(resp.AudioContent))
	return resp.AudioContent, nil
}

// UploadAudioToS3 mengunggah file audio ke S3 agar bisa diproses oleh Transcribe
func UploadUserVoiceToS3(fileName string, audio []byte) (string, error) {
	uploader := manager.NewUploader(s3Client)

	key := "user/" + fileName

	upInput := &s3.PutObjectInput{
		Bucket:      aws.String(config.AWSBucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(audio),
		ContentType: aws.String("audio/mpeg"),
	}

	_, err := uploader.Upload(context.TODO(), upInput)
	if err != nil {
		return "", fmt.Errorf("gagal mengunggah suara pengguna ke S3: %v", err)
	}

	audioURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", config.AWSBucketName, config.AWSRegion, key)
	return audioURL, nil
}

func UploadBotVoiceToS3(fileName string, audio []byte) (string, error) {
	uploader := manager.NewUploader(s3Client)

	key := "bot/" + fileName

	upInput := &s3.PutObjectInput{
		Bucket:      aws.String(config.AWSBucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(audio),
		ContentType: aws.String("audio/mpeg"),
	}

	_, err := uploader.Upload(context.TODO(), upInput)
	if err != nil {
		return "", fmt.Errorf("gagal mengunggah suara bot ke S3: %v", err)
	}

	audioURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", config.AWSBucketName, config.AWSRegion, key)
	return audioURL, nil
}

func TranscribeAudio(audioData []byte) (string, string, error) {
	audioFileName := fmt.Sprintf("user-audio-%d.mp3", time.Now().Unix())
	log.Printf("🔼 Mengunggah audio pengguna ke S3: %s", audioFileName)

	// ✅ Upload audio user ke S3
	audioURL, err := UploadUserVoiceToS3(audioFileName, audioData)
	if err != nil {
		log.Printf("❌ Gagal mengunggah audio pengguna ke S3: %v", err)
		return "", "", fmt.Errorf("failed to upload audio: %v", err)
	}
	log.Printf("✅ Audio pengguna berhasil diunggah ke S3, URI: %s", audioURL)

	jobName := fmt.Sprintf("transcribe-job-%d", time.Now().Unix())
	log.Printf("🎙️ Mulai tugas transkripsi: %s", jobName)

	// ✅ Kirim S3 URI ke AWS Transcribe
	err = StartTranscriptionJob(jobName, audioURL)
	if err != nil {
		log.Printf("❌ Gagal memulai tugas transkripsi: %v", err)
		return "", "", fmt.Errorf("failed to start transcription job: %v", err)
	}

	log.Printf("⌛ Menunggu hasil transkripsi untuk tugas: %s", jobName)
	transcript, err := GetTranscriptionResult(jobName)
	if err != nil {
		return "", "", err
	}

	return transcript, audioURL, nil
}

// StartTranscriptionJob memulai proses transkripsi dengan Amazon Transcribe
func StartTranscriptionJob(jobName, mediaUri string) error {
	input := &transcribe.StartTranscriptionJobInput{
		TranscriptionJobName: aws.String(jobName),
		LanguageCode:         transcribeTypes.LanguageCodeIdId,
		MediaFormat:          transcribeTypes.MediaFormatMp3,
		Media: &transcribeTypes.Media{
			MediaFileUri: aws.String(mediaUri),
		},
		OutputBucketName: aws.String(config.AWSBucketName),
	}

	_, err := transcribeClient.StartTranscriptionJob(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("gagal memulai tugas transkripsi: %v", err)
	}
	return nil
}

// GetTranscriptionResult mengambil hasil transkripsi jika sudah selesai
func GetTranscriptionResult(jobName string) (string, error) {
	for i := 0; i < 20; i++ {
		job, err := transcribeClient.GetTranscriptionJob(context.TODO(), &transcribe.GetTranscriptionJobInput{
			TranscriptionJobName: aws.String(jobName),
		})
		if err != nil {
			return "", fmt.Errorf("gagal mendapatkan tugas transkripsi: %v", err)
		}

		status := job.TranscriptionJob.TranscriptionJobStatus
		if status == transcribeTypes.TranscriptionJobStatusCompleted {
			// Ambil URL hasil transkripsi
			transcriptUrl := *job.TranscriptionJob.Transcript.TranscriptFileUri
			return fetchTranscriptFromURL(transcriptUrl)
		} else if status == transcribeTypes.TranscriptionJobStatusFailed {
			return "", errors.New("tugas transkripsi gagal")
		}

		time.Sleep(3 * time.Second) // tunggu sebelum cek ulang
	}

	return "", errors.New("tugas transkripsi kehabisan waktu")
}

// fetchTranscriptFromURL mengambil hasil transkrip dari URL (berformat JSON)
func fetchTranscriptFromURL(url string) (string, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("gagal mengambil transkrip: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("url transkrip mengembalikan status non-OK: %s", resp.Status)
	}

	var result struct {
		Results struct {
			Transcripts []struct {
				Transcript string `json:"transcript"`
			} `json:"transcripts"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("gagal mendekode respons transkrip: %v", err)
	}

	if len(result.Results.Transcripts) == 0 {
		return "", errors.New("tidak ditemukan transkrip dalam hasil")
	}

	return result.Results.Transcripts[0].Transcript, nil
}

func SaveVoiceChatHistory(chatID string, userID int, transcript, intent, userAudioURL, botAudioURL, responseMessage string) error {
	collection := config.MongoDB.Collection("voice_messages")

	now := time.Now()

	userMsg := models.VoiceMessage{
		ChatID:     chatID,
		UserID:     userID,
		Sender:     "user",
		AudioURL:   userAudioURL,
		Transcript: transcript,
		Intent:     intent,
		Timestamp:  now,
	}

	botMsg := models.VoiceMessage{
		ChatID:     chatID,
		UserID:     userID,
		Sender:     "bot",
		AudioURL:   botAudioURL,
		Transcript: responseMessage,
		Intent:     intent,
		Timestamp:  now.Add(1 * time.Millisecond),
	}

	_, err := collection.InsertMany(context.TODO(), []interface{}{userMsg, botMsg})
	return err
}
