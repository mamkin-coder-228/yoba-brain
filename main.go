package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/vsergeev/btckeygenie/btckey"
)

var total uint64
var generated uint64
var success uint64

type wItem struct {
	Wif            string
	AddrUncompress string
	AddrCompress   string
}

func Generate() []wItem {
	var wallets []wItem
	for i := 0; i < 32; i++ {
		priv, _ := btckey.GenerateKey(rand.Reader)
		currWallet := wItem{
			Wif:            priv.ToWIF(),
			AddrUncompress: priv.ToAddressUncompressed(),
			AddrCompress:   priv.ToAddress(),
		}
		atomic.AddUint64(&generated, 1)
		wallets = append(wallets, currWallet)
	}
	return wallets
}

func Check(wallets []wItem) {
	var toSend string
	for _, cur := range wallets {
		toSend = toSend + fmt.Sprintf("%s|", cur.AddrUncompress)
	}
	url := fmt.Sprintf("https://blockchain.info/balance?cors=true&active=%s", toSend)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}

	for {
		if resp.StatusCode == 200 {
			break
		} else {
			fmt.Printf("Ошибка при получении ответа от сервера 200!=%v - повторяем запрос\n", resp.StatusCode)
			time.Sleep(time.Second)
			resp, err = http.Get(url)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	for id, cur := range result {
		c := cur.(map[string]interface{})
		final_balance, _ := c["final_balance"]
		n_tx, _ := c["n_tx"]
		total_received, _ := c["n_tx"]

		if final_balance.(float64) != 0 {
			for _, curW := range wallets {
				if curW.AddrUncompress == id {
					atomic.AddUint64(&success, 1)

					log.Printf("%v - Address: %f | %v | %v -> WIF: %v\n", id, final_balance, n_tx, total_received, curW.Wif)
					break
				}
			}

		}
		atomic.AddUint64(&total, 1)
	}
}

func checkLoop() {
	for {
		data := Generate()
		Check(data)
	}
}

func main() {
	fmt.Println("YOBA BRAIN v0.2")
	fmt.Println("Ну че, народ, погнали, наxуй!")

	startTime := time.Now()
	logFile, err := os.OpenFile("goods.txt", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	var threadsCount int
	threadsCount = runtime.NumCPU()
	fmt.Printf("Ебошим в %v потоков\n", threadsCount)
	for i := 0; i < threadsCount; i++ {
		go checkLoop()
	}

	for {
		time.Sleep(time.Second * 10)
		opsFinal := atomic.LoadUint64(&total)
		genFinal := atomic.LoadUint64(&generated)
		speedCheck := opsFinal / uint64(time.Since(startTime).Seconds())
		speedGen := genFinal / uint64(time.Since(startTime).Seconds())
		fmt.Printf("[%v] Сгенерированно: %v | Проверено: %v | Найдено: %v | Скорость проверки (кошелей в сек): %v | Скорость генерирования (кошелей в сек): %v\n", time.Now().Format("Mon Jan 2 15:04:05 MST 2006"), genFinal, opsFinal, atomic.LoadUint64(&success), speedCheck, speedGen)
	}
}
