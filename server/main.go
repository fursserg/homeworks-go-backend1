package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

type client chan<- string

var (
	entering     = make(chan client)
	leaving      = make(chan client)
	messages     = make(chan string)
	clientsNicks = make(map[string]string)
)

func main() {
	listener, err := net.Listen("tcp", "localhost:8000")

	if err != nil {
		log.Fatal(err)
	}

	go broadcaster()
	go serverMessagesSender()

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Print(err)
			continue
		}

		go handleConn(conn)
	}
}

// ДЗ 1. Слушаем ввод на сервере и отправляем его в канал для последующей рассылки клиентам
func serverMessagesSender() {
	input := bufio.NewScanner(os.Stdin)

	for input.Scan() {
		messages <- "message from server: " + input.Text()
	}
}

func broadcaster() {
	clients := make(map[client]bool)

	for {
		select {
		case msg := <-messages:
			for cli := range clients {
				cli <- msg
			}
		case cli := <-entering:
			clients[cli] = true
		case cli := <-leaving:
			delete(clients, cli)
			close(cli)
		}
	}
}

func handleConn(conn net.Conn) {
	ch := make(chan string)
	nick := ""
	var wantedNick string

	go clientWriter(conn, ch)

	ip := conn.RemoteAddr().String()
	ch <- "Hi! Type your nickname, please: "
	entering <- ch

	input := bufio.NewScanner(conn)

	for input.Scan() {
		// ДЗ 2. В первом сообщении от клиента ожидаем его ник
		if nick == "" {
			wantedNick = input.Text()

			// ДЗ 2. Если клиент вводит не уникальный ник, то перезапрашиваем
			if setNickName(wantedNick, ip) {
				nick = wantedNick
				messages <- nick + " has arrived"
			} else {
				ch <- fmt.Sprintf("Sorry, but the nickname '%s' is already in use, try another one: ", wantedNick)
			}

			continue
		}

		messages <- nick + ": " + input.Text()
	}

	leaving <- ch
	messages <- nick + " has left"
	removeNickName(nick)

	conn.Close()
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg)
	}
}

func setNickName(nick string, ip string) bool {
	if _, ok := clientsNicks[nick]; ok {
		return false
	}

	clientsNicks[nick] = ip
	return true
}

func removeNickName(nick string) {
	delete(clientsNicks, nick)
}
