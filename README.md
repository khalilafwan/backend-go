# ⚙️ RANCANG BANGUN CHATBOT MULTIMODAL – BACKEND

Backend untuk sistem chatbot multimodal yang dikembangkan sebagai bagian dari tugas akhir berjudul  
**“Rancang Bangun Fitur Chatbot Multimodal Terintegrasi Teks dan Suara untuk Meningkatkan Pelayanan Nasabah di Bank Nagari.”**

---

## 🚀 Teknologi Utama

- 🧩 **Go (Gin Framework)** – REST API utama untuk komunikasi antara frontend dan server.  
- 🗄️ **MongoDB** – Penyimpanan data percakapan dan pengguna.  
- 🔐 **JWT Authentication** – Sistem autentikasi dan otorisasi pengguna.  
- 🧠 **Flask NLP API** – Proses intent recognition & natural language understanding.  
- 🔊 **AWS Polly & AWS Transcribe** – Fitur Text-to-Speech (TTS) dan Speech-to-Text (STT).  
- ☁️ **AWS S3** – Penyimpanan file audio hasil voice mode.  

---

## 💡 Fitur Utama

- 🔑 Registrasi dan login pengguna dengan JWT.  
- 💬 Endpoint percakapan berbasis teks dan suara.  
- 🧠 Integrasi NLP Flask untuk deteksi intent dan respons otomatis.  
- 🎙️ Fitur *voice mode* (chat dengan suara menggunakan AWS Polly & Transcribe).  
- 💾 Penyimpanan riwayat chat dan metadata ke MongoDB.  
- 🗣️ Dukungan *voice change* (ubah gender suara TTS).  

---

## ⚙️ Cara Menjalankan

Pastikan MongoDB dan Flask NLP sudah berjalan sebelum menjalankan backend.

```bash
# Clone repository
git clone https://github.com/username/backend-go.git
cd backend-go

# Install dependency
go mod tidy

# Jalankan server
go run main.go
