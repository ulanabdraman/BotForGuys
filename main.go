package main

import (
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

var (
	subscribers = make(map[int64]bool) // 🔹 Список подписанных пользователей
	mu          sync.Mutex             // 🔐 Защита от гонок данных
)

func main() {
	// Загружаем переменные из .env файла
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка при загрузке .env файла")
	}

	// Получаем значения переменных окружения
	TelegramToken := os.Getenv("TELEGRAM_TOKEN")
	SecretWord := os.Getenv("SECRET_WORD")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Значение по умолчанию для локального запуска
	}
	APIListenAddr := ":" + port
	log.Println("все успешно", TelegramToken, SecretWord, APIListenAddr)
	bot, err := tgbotapi.NewBotAPI(TelegramToken)
	if err != nil {
		log.Fatal("Ошибка при создании бота:", err)
	}

	go startTelegramBot(bot, SecretWord)     // 🔄 Запускаем бота в отдельной горутине
	startAPI(bot, SecretWord, APIListenAddr) // 🌍 Запускаем API
}

// 📩 Функция обработки сообщений в Telegram
func startTelegramBot(bot *tgbotapi.BotAPI, SecretWord string) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // Игнорируем не-текстовые сообщения
			continue
		}

		chatID := update.Message.Chat.ID
		text := update.Message.Text

		if text == SecretWord { // 🎯 Проверяем секретное слово
			mu.Lock()
			subscribers[chatID] = true
			mu.Unlock()

			msg := tgbotapi.NewMessage(chatID, "✅ Вы подписались на оповещения!")
			bot.Send(msg)
		}
	}
}

// 🌍 Запуск API-сервера
func startAPI(bot *tgbotapi.BotAPI, SecretWord, APIListenAddr string) {
	r := gin.Default()

	// 🔥 API для отправки сообщений подписчикам
	r.POST("/notify", func(c *gin.Context) {
		var req struct {
			Message string `json:"message"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
			return
		}

		mu.Lock()
		defer mu.Unlock()

		for chatID := range subscribers {
			msg := tgbotapi.NewMessage(chatID, req.Message)
			bot.Send(msg)
		}

		c.JSON(http.StatusOK, gin.H{"status": "Оповещение отправлено"})
	})

	log.Println("📡 API запущен на", APIListenAddr)
	r.Run("0.0.0.0" + APIListenAddr)
}
