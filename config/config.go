package config

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/spf13/viper"
)

var (
	Environment            string
	SessionSecret          string
	PostgresUser           string
	PostgresPassword       string
	PostgresDB             string
	PostgresHost           string
	PostgresPort           string
	PythonNLPURL           string
	Port                   string
	AWSRegion              string
	AWSBucketName          string
	AWSAccessKeyID         string
	AWSSecretAccessKey     string
	GoogleApplicationCreds string

	// singleton lock
	loadConfigOnce sync.Once
)

var AWSConfig aws.Config

// LoadConfig loads configuration from .env or config.yaml using Viper
func LoadConfig() error {
	var loadError error
	loadConfigOnce.Do(func() {
		// Try to load config from .env first, then fallback to config.yaml
		viper.SetConfigFile(".env")
		viper.AutomaticEnv()

		if err := viper.ReadInConfig(); err != nil {
			viper.SetConfigFile("config.yaml")
			if err := viper.ReadInConfig(); err != nil {
				loadError = err // Store the error to be returned
				log.Println("Gagal memuat file konfigurasi:", err)
				return
			}
		}

		// Assign variables from configuration
		PostgresUser = viper.GetString("POSTGRES_USER")
		PostgresPassword = viper.GetString("POSTGRES_PASSWORD")
		PostgresDB = viper.GetString("POSTGRES_DB")
		PostgresHost = viper.GetString("POSTGRES_HOST")
		PostgresPort = viper.GetString("POSTGRES_PORT")
		PythonNLPURL = viper.GetString("PYTHON_NLP_URL")
		Port = viper.GetString("PORT")
		Environment = viper.GetString("ENVIRONMENT")
		SessionSecret = viper.GetString("SESSION_SECRET")
		AWSAccessKeyID = viper.GetString("AWS_ACCESS_KEY_ID")
		AWSSecretAccessKey = viper.GetString("AWS_SECRET_ACCESS_KEY")
		AWSRegion = viper.GetString("AWS_REGION")
		AWSBucketName = viper.GetString("AWS_BUCKET_NAME")
		GoogleApplicationCreds = viper.GetString("GOOGLE_APPLICATION_CREDENTIALS")

		// Set environment var for Google Cloud SDK
		if GoogleApplicationCreds == "" {
			log.Println("⚠️ GOOGLE_APPLICATION_CREDENTIALS belum diatur")
		} else {
			err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", GoogleApplicationCreds)
			if err != nil {
				log.Fatalf("❌ Gagal mengatur GOOGLE_APPLICATION_CREDENTIALS: %v", err)
			}
			log.Println("✅ Kredensial Google Cloud telah dikonfigurasi")
		}

		log.Println("✅ Konfigurasi berhasil dimuat!")
	})

	return loadError
}

func LoadAWSConfig() error {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(AWSRegion),
		config.WithCredentialsProvider(
			aws.NewCredentialsCache(
				credentials.NewStaticCredentialsProvider(AWSAccessKeyID, AWSSecretAccessKey, ""),
			),
		),
	)
	if err != nil {
		return err
	}
	AWSConfig = cfg
	log.Println("✅ Konfigurasi AWS SDK berhasil dimuat (manual credentials)")
	log.Printf("📦 Menggunakan wilayah AWS: %s", cfg.Region)
	return nil
}
