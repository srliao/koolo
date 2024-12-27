package discord

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/hectorgimenez/koolo/internal/bot"
	"github.com/hectorgimenez/koolo/internal/config"
)

type Bot struct {
	discordSession *discordgo.Session
	channelID      string
	manager        *bot.SupervisorManager
	logger         *slog.Logger
}

func NewBot(token, channelID string, manager *bot.SupervisorManager, logger *slog.Logger) (*Bot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	return &Bot{
		discordSession: dg,
		channelID:      channelID,
		manager:        manager,
		logger:         logger,
	}, nil
}

func (b *Bot) Start(ctx context.Context) error {
	//b.discordSession.Debug = true
	b.discordSession.AddHandler(b.onMessageCreated)
	b.discordSession.Identify.Intents = discordgo.IntentsGuildMessages
	err := b.discordSession.Open()
	if err != nil {
		b.logger.Warn("error opening connection", "err", err)
		return fmt.Errorf("error opening connection: %w", err)
	}
	b.logger.Info("discord bot connected ok")

	// Wait until context is finished
	<-ctx.Done()

	return b.discordSession.Close()
}

func (b *Bot) onMessageCreated(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if the message is from a bot admin
	if !slices.Contains(config.Koolo.Discord.BotAdmins, m.Author.ID) {
		b.logger.Info("message received but ignored, unrecognized user", "admins", config.Koolo.Discord.BotAdmins, "sender", m.Author.ID)
		return
	}

	prefix := strings.Split(m.Content, " ")[0]
	b.logger.Info("message received", "prefix", prefix, "content", m.Content)
	switch prefix {
	case "!start":
		b.handleStartRequest(s, m)
	case "!stop":
		b.handleStopRequest(s, m)
	case "!stats":
		b.handleStatsRequest(s, m)
	case "!status":
		b.handleStatusRequest(s, m)
	}

}
