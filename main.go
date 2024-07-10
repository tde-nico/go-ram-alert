package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/shirou/gopsutil/mem"
)

type ConfigStruct struct {
	file    string
	DSToken string `json:"DSToken"`
	ChatID  string `json:"ChatID"`
}

var config ConfigStruct

var timeout int = 60
var limit uint64 = 1024 * 1024 * 1024 * 7
var percent float64 = 0

func usageMonitorLoop(s *discordgo.Session) {
	for {
		vm, err := mem.VirtualMemory()
		if err != nil {
			log.Println("Failed to get memory information:", err)
			return
		}

		if percent != 0 && vm.UsedPercent > percent {
			log.Printf("Error: to much ram -> %v\n", vm.UsedPercent)
			_, err := s.ChannelMessageSend(config.ChatID, fmt.Sprintf("Error: to much ram -> %v%%", vm.UsedPercent))
			if err != nil {
				log.Println("Failed to send message:", err)
			}

		} else if vm.Used > limit {
			log.Printf("Error: to much ram -> %v/%v\n", vm.Used, vm.Total)
			_, err := s.ChannelMessageSend(config.ChatID, fmt.Sprintf("Error: to much ram -> %v%%", vm.UsedPercent))
			if err != nil {
				log.Println("Failed to send message:", err)
			}
		}

		log.Printf("RAM: %v/%v -> %v\n", vm.Used, vm.Total, vm.UsedPercent)
		time.Sleep(time.Duration(timeout) * time.Second)
	}

}

func (c *ConfigStruct) Load(fname string) {
	c.file = fname
	file, err := os.ReadFile(fname)
	if err != nil {
		log.Fatalf("error reading file %v: %v\n", fname, err)
	}
	if err = json.Unmarshal(file, c); err != nil {
		log.Fatalf("error unmarshalling JSON %v: %v\n", fname, err)
	}
}

func (c *ConfigStruct) Save() {
	file, err := os.OpenFile(c.file, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("error opening file %v: %v\n", c.file, err)
		return
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "    ")
	if err = enc.Encode(c); err != nil {
		log.Printf("error encoding JSON: %v\n", err)
	}
}

func main() {
	flag.IntVar(&timeout, "t", timeout, "timeout in seconds")
	flag.Float64Var(&percent, "p", percent, "limit in percent")
	flag.Uint64Var(&limit, "l", limit, "limit in bytes")
	flag.Parse()

	config.Load("./config.json")

	session, err := discordgo.New("Bot " + config.DSToken)
	if err != nil {
		log.Fatalf("error creating Discord session: %v\n", err)
	}
	defer session.Close()

	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	err = session.Open()
	if err != nil {
		log.Fatalf("error opening connection: %v\n", err)
	}

	go usageMonitorLoop(session)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Bot is now running. Press CTRL-C to exit.")
	<-stop
}
