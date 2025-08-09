package main

import (
	"emote-counter/internal/database"
	"emote-counter/internal/handlers"
	"emote-counter/internal/utils"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"gorm.io/gorm/clause"
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
	for _, guild := range guildIds {
		emotes, err := utils.GetGuildEmotes(sess, guild)
		if err != nil {
			fmt.Println(err)
			continue
		}

		var emoteCounts []database.EmoteCount
		for _, emote := range emotes {
			emoteCounts = append(emoteCounts, database.EmoteCount{
				GuildID:  guild,
				ID:       emote.ID,
				Name:     emote.Name,
				Animated: emote.Animated,
			})
			log.Println(emote.Animated)
		}

		var added []database.EmoteCount
		if len(emotes) > 0 {
			database.DB.Clauses(clause.OnConflict{DoNothing: true}, clause.Returning{}).
				Create(&emoteCounts).Scan(&added)
		}
		if len(added) > 0 {
			log.Printf("Updated emotes for guild '%v'", guild)
		}
	}

	log.Println("Logged as " + sess.State.User.Username + "#" + sess.State.User.Discriminator)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
