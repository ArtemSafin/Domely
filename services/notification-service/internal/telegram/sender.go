package telegram

import (
	"fmt"

	tele "gopkg.in/telebot.v3"
)

const (
	priorityHigh   = "high"
	priorityNormal = "normal"
	priorityLow    = "low"
)

type Sender struct {
	bot *tele.Bot
}

func NewSender(bot *tele.Bot) *Sender {
	return &Sender{bot: bot}
}

func (s *Sender) SendReminder(telegramID int64, title, priority string) error {
	recipient := &tele.User{ID: telegramID}

	text := formatMessage(title, priority)

	if _, err := s.bot.Send(recipient, text, tele.ModeMarkdown); err != nil {
		return fmt.Errorf("send message to %d: %w", telegramID, err)
	}

	return nil
}

func formatMessage(title, priority string) string {
	icon := priorityIcon(priority)
	return fmt.Sprintf("%s *Напоминание*\n\n%s", icon, title)
}

func priorityIcon(priority string) string {
	switch priority {
	case priorityHigh:
		return "🔴"
	case priorityLow:
		return "🟢"
	default:
		return "🟡"
	}
}
