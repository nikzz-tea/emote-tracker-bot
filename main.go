package main

import (
	"emote-counter/internal/database"
	"emote-counter/internal/handlers"
	"emote-counter/internal/utils"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "emote-counter/internal/commands"
	_ "emote-counter/internal/events"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	token := os.Getenv("TOKEN")
	guilds := os.Getenv("GUILDS")

	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}

	sess.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = sess.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer sess.Close()

	sess.AddHandler(handlers.CommandHandler)
	handlers.EventHandler(sess)

	database.Init()

	guildIds := strings.Split(guilds, ",")

	syncCommands(sess, guildIds)
	syncGuildEmotes(sess, guildIds)

	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			syncGuildEmotes(sess, guildIds)
		}
	}()

	log.Println("Logged as " + sess.State.User.Username + "#" + sess.State.User.Discriminator)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func syncCommands(sess *discordgo.Session, guildIds []string) {
	for _, cmd := range handlers.GetCommands() {
		appCmd := &discordgo.ApplicationCommand{
			Name:        cmd.Name,
			Description: cmd.Description,
			Options:     cmd.Options,
		}
		for _, guildID := range guildIds {
			_, err := sess.ApplicationCommandCreate(sess.State.User.ID, guildID, appCmd)
			if err != nil {
				log.Printf("Cannot create '/%v' command for guild %v: %v", cmd.Name, guildID, err)
			}
		}
	}
}

func syncGuildEmotes(sess *discordgo.Session, guildIds []string) {
	for _, guild := range guildIds {
		emotes, err := utils.GetGuildEmotes(sess, guild)
		if err != nil {
			log.Println("Error fetching emotes for guild", guild, ":", err)
			continue
		}
		log.Printf("Guild '%v' has %d emotes available", guild, len(emotes))
	}
}
