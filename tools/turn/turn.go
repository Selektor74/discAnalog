package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/pion/turn/v2"
)

func main() {
	defaultPublicIP := strings.TrimSpace(os.Getenv("TURN_PUBLIC_IP"))
	defaultUsers := strings.TrimSpace(os.Getenv("TURN_USERS"))
	defaultRealm := strings.TrimSpace(os.Getenv("TURN_REALM"))
	if defaultRealm == "" {
		defaultRealm = "selektor"
	}

	publicIP := flag.String("public-ip", defaultPublicIP, "")
	port := flag.Int("port", 3478, "")
	users := flag.String("users", defaultUsers, "")
	realm := flag.String("realm", defaultRealm, "")
	flag.Parse()

	if len(*publicIP) == 0 {
		log.Fatalf("Не указан public-ip (turn)")
	}

	if len(*users) == 0 {
		log.Fatalf("users не указаны (turn)")
	}

	udpListener, err := net.ListenPacket("udp4", "0.0.0.0:"+strconv.Itoa(*port))

	if err != nil {
		log.Panicf("Не удалось создать TURN сервер :%s", err)
	}

	usersMap := map[string][]byte{}

	for _, pair := range strings.Split(*users, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		username := strings.TrimSpace(kv[0])
		password := strings.TrimSpace(kv[1])
		if username == "" || password == "" {
			continue
		}
		usersMap[username] = turn.GenerateAuthKey(username, *realm, password)
	}

	if len(usersMap) == 0 {
		log.Fatalf("users не указаны или имеют неверный формат, ожидается user=password[,user2=password2]")
	}

	s, err := turn.NewServer(turn.ServerConfig{
		Realm: *realm,
		AuthHandler: func(username string, realm string, scrAddr net.Addr) ([]byte, bool) {
			if key, ok := usersMap[username]; ok {
				return key, true
			}
			return nil, false
		},

		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorPortRange{
					RelayAddress: net.ParseIP(*publicIP),
					Address:      "0.0.0.0",
					MinPort:      50000,
					MaxPort:      55000,
				},
			},
		},
	})

	if err != nil {
		log.Panic(err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	if err = s.Close(); err != nil {
		log.Panic(err)
	}
}
