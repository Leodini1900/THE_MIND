package main

import (
    "log"
    "math/rand"
    "net/http"
    "strconv"
    "sync"
    "time"

    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

type Game struct {
    Deck              []int
    Colors            []string
    Mutex             sync.Mutex
    Played            []Card
    Players           map[*websocket.Conn]Player
    Level             int
    Points            int
    Lives             int
    Shurikens         int
    ConcentrationMode map[*websocket.Conn]bool
}

type Card struct {
    Number int
    Color  string
}

type Player struct {
    Cards    []Card
    Ready    bool
    PlayerId int
}

var game = Game{
    Deck:              generateDeck(),
    Colors:            []string{"skyblue", "red", "yellow", "orange", "purple", "pink", "brown", "lightgreen", "lightcoral", "lightseagreen"},
    Played:            []Card{},
    Players:           make(map[*websocket.Conn]Player),
    Level:             1,
    Points:            0,
    Lives:             3,
    Shurikens:         1,
    ConcentrationMode: make(map[*websocket.Conn]bool),
}

func generateDeck() []int {
    deck := make([]int, 100)
    for i := 0; i < 100; i++ {
        deck[i] = i + 1
    }
    rand.Seed(time.Now().UnixNano())
    rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
    return deck
}

func main() {
    http.HandleFunc("/ws", handleConnections)
    log.Println("Server started on :8085")
    log.Fatal(http.ListenAndServe(":8085", nil))
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("Upgrade error:", err)
        return
    }
    defer conn.Close()

    game.Mutex.Lock()
    if len(game.Deck) == 0 || len(game.Players) >= 10 {
        game.Mutex.Unlock()
        conn.WriteJSON(map[string]string{
            "type":    "status",
            "message": "Jeu complet ou plus de cartes disponibles",
        })
        return
    }
    dealCards(conn)
    game.ConcentrationMode[conn] = false
    game.Mutex.Unlock()
    broadcastPlayersUpdate()

    for {
        var msg map[string]interface{}
        err := conn.ReadJSON(&msg)
        if err != nil {
            log.Println("ReadJSON error:", err)
            game.Mutex.Lock()
            delete(game.Players, conn)
            delete(game.ConcentrationMode, conn)
            game.Mutex.Unlock()
            broadcastPlayersUpdate()
            broadcastConcentrationStatus()
            break
        }

        log.Printf("Received message: %v\n", msg)

        if msg["type"] == "play" {
            cardNumber, ok := msg["card"].(float64)
            if !ok {
                log.Println("Invalid card type")
                continue
            }
            cardColor, ok := msg["color"].(string)
            if !ok {
                log.Println("Invalid color type")
                continue
            }

            game.Mutex.Lock()
            if len(game.Played) > 0 && int(cardNumber) < game.Played[len(game.Played)-1].Number {
                game.Lives--
                if game.Lives <= 0 {
                    game.Mutex.Unlock()
                    broadcastStatus("Perdu! Les cartes n'ont pas été jouées dans l'ordre croissant. Plus de vies restantes.")
                    resetGame()
                } else {
                    broadcastStatus("Erreur! Les cartes n'ont pas été jouées dans l'ordre croissant. Vies restantes: " + strconv.Itoa(game.Lives))
                    reshuffleDeck()
                    broadcastClearTable()
                    distributeNewCards()
                    game.Mutex.Unlock()
                }
                continue
            }
            game.Played = append(game.Played, Card{Number: int(cardNumber), Color: cardColor})
            game.Mutex.Unlock()
            broadcastPlay(int(cardNumber), cardColor)

            if len(game.Played) == game.Level*len(game.Players) {
                game.Mutex.Lock()
                game.Points += len(game.Players)
                if game.Level >= 10 {
                    broadcastStatus("Jeu gagné! Félicitations! Points: " + strconv.Itoa(game.Points))
                    resetGame()
                } else {
                    game.Level++
                    broadcastStatus("Niveau " + strconv.Itoa(game.Level) + " terminé! Points: " + strconv.Itoa(game.Points))
                    reshuffleDeck()
                    broadcastClearTable()
                    distributeNewCards()
                }
                game.Mutex.Unlock()
            }
        }

        if msg["type"] == "concentration" {
            game.Mutex.Lock()
            playerConcentration, ok := msg["concentration"].(bool)
            if ok {
                game.ConcentrationMode[conn] = playerConcentration
                broadcastConcentrationStatus()
            }
            game.Mutex.Unlock()
        }

        if msg["type"] == "shuriken" {
            game.Mutex.Lock()
            if game.Shurikens > 0 {
                game.Shurikens--
                for conn, player := range game.Players {
                    if len(player.Cards) > 0 {
                        smallestCard := player.Cards[0]
                        for _, card := range player.Cards {
                            if card.Number < smallestCard.Number {
                                smallestCard = card
                            }
                        }
                        game.Played = append(game.Played, smallestCard)
                        player.Cards = removeCard(player.Cards, smallestCard)
                        game.Players[conn] = player
                        conn.WriteJSON(map[string]interface{}{
                            "type":  "update",
                            "cards": player.Cards,
                        })
                    }
                }
                broadcastStatus("Shuriken utilisé! Cartes les plus petites défaussées.")
                broadcastClearTable()
                distributeNewCards()
            } else {
                conn.WriteJSON(map[string]string{
                    "type":    "status",
                    "message": "Pas de shurikens restants",
                })
            }
            game.Mutex.Unlock()
        }

        if msg["type"] == "ready" {
            game.Mutex.Lock()
            player, ok := game.Players[conn]
            if ok {
                player.Ready = !player.Ready
                game.Players[conn] = player
                broadcastPlayersUpdate()
            }
            game.Mutex.Unlock()
        }
    }
}

func dealCards(conn *websocket.Conn) {
    playerCards := make([]Card, game.Level)
    for i := 0; i < game.Level; i++ {
        playerCard := game.Deck[0]
        game.Deck = game.Deck[1:]
        playerCards[i] = Card{Number: playerCard, Color: game.Colors[len(game.Players)%len(game.Colors)]}
    }
    playerId := len(game.Players)
    game.Players[conn] = Player{Cards: playerCards, Ready: false, PlayerId: playerId}
    conn.WriteJSON(map[string]interface{}{
        "type":    "init",
        "cards":   playerCards,
        "level":   game.Level,
        "lives":   game.Lives,
        "shurikens": game.Shurikens,
        "playerId": playerId,
        "players": getPlayersData(),
    })
}

func broadcastPlay(cardNumber int, cardColor string) {
    msg := map[string]interface{}{
        "type":  "play",
        "card":  cardNumber,
        "color": cardColor,
    }
    for client := range game.Players {
        err := client.WriteJSON(msg)
        if err != nil {
            log.Println("WriteJSON error:", err)
            client.Close()
            delete(game.Players, client)
            delete(game.ConcentrationMode, client)
        }
    }
}

func broadcastStatus(message string) {
    msg := map[string]string{
        "type":    "status",
        "message": message,
    }
    for client := range game.Players {
        err := client.WriteJSON(msg)
        if err != nil {
            log.Println("WriteJSON error:", err)
            client.Close()
            delete(game.Players, client)
            delete(game.ConcentrationMode, client)
        }
    }
}

func broadcastClearTable() {
    msg := map[string]string{
        "type": "clear",
    }
    for client := range game.Players {
        err := client.WriteJSON(msg)
        if err != nil {
            log.Println("WriteJSON error:", err)
            client.Close()
            delete(game.Players, client)
            delete(game.ConcentrationMode, client)
        }
    }
}

func broadcastConcentrationStatus() {
    allInConcentration := true
    for _, status := range game.ConcentrationMode {
        if !status {
            allInConcentration = false
            break
        }
    }

    concentrationStatus := "concentration"
    if allInConcentration {
        concentrationStatus = "game"
    }

    msg := map[string]string{
        "type":   "concentration_status",
        "status": concentrationStatus,
    }
    for client := range game.Players {
        err := client.WriteJSON(msg)
        if err != nil {
            log.Println("WriteJSON error:", err)
            client.Close()
            delete(game.Players, client)
            delete(game.ConcentrationMode, client)
        }
    }
}

func broadcastPlayersUpdate() {
    msg := map[string]interface{}{
        "type":   "players_update",
        "players": getPlayersData(),
    }
    for client := range game.Players {
        err := client.WriteJSON(msg)
        if err != nil {
            log.Println("WriteJSON error:", err)
            client.Close()
            delete(game.Players, client)
            delete(game.ConcentrationMode, client)
        }
    }
}

func getPlayersData() map[int]map[string]bool {
    playersData := make(map[int]map[string]bool)
    for _, player := range game.Players {
        playersData[player.PlayerId] = map[string]bool{
            "connected": true,
            "ready":     player.Ready,
        }
    }
    return playersData
}

func resetGame() {
    game.Deck = generateDeck()
    game.Played = []Card{}
    game.Players = make(map[*websocket.Conn]Player)
    game.Level = 1
    game.Points = 0
    game.Lives = 3
    game.Shurikens = 1
    game.ConcentrationMode = make(map[*websocket.Conn]bool)
}

func reshuffleDeck() {
    for _, card := range game.Played {
        game.Deck = append(game.Deck, card.Number)
    }
    game.Played = []Card{}
    rand.Shuffle(len(game.Deck), func(i, j int) { game.Deck[i], game.Deck[j] = game.Deck[j], game.Deck[i] })
}

func distributeNewCards() {
    for conn := range game.Players {
        dealCards(conn)
    }
}

func removeCard(cards []Card, card Card) []Card {
    for i, c := range cards {
        if c.Number == card.Number {
            return append(cards[:i], cards[i+1:]...)
        }
    }
    return cards
}
