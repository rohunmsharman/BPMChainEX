package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net"

	"strconv"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"

)

type Block struct {
	Index     int
	Timestamp string
	BPM       int
	Hash      string
	PrevHash  string
}

// blockchain
var Blockchain []Block
var tempBlocks []Block
// bcServer handles incoming Blocks
var bcServer chan []Block
var mutex = &sync.Mutex{}

func main() {

	bcServer = make(chan []Block)

	// create genesis block
	t := time.Now()
	genesisBlock := Block{0, t.String(), 0, "", ""}
	spew.Dump(genesisBlock)
	Blockchain = append(Blockchain, genesisBlock)

	httpPort := ":9000"

	// start TCP and serve TCP server
	server, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("listening  on port :", httpPort)
	defer server.Close()

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConn(conn)
	}

}

func handleConn(conn net.Conn) {

	defer conn.Close()

	io.WriteString(conn, "Enter send amount")

	scanner := bufio.NewScanner(conn)

	// take in BPM from stdin and add it to blockchain after validation
	go func() {
		for scanner.Scan() {
			bpm, err := strconv.Atoi(scanner.Text())
			if err != nil {
				log.Printf("%v not a number: %v", scanner.Text(), err)
				continue
			}
			newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], bpm)
			if err != nil {
				log.Println(err)
				continue
			}
			if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
				newBlockchain := append(Blockchain, newBlock)
				replaceChain(newBlockchain)
			}

			bcServer <- Blockchain
			io.WriteString(conn, "\nEnter a new BPM:")
		}
	}()

	// pseudo receiving broadcast
	go func() {
		for {
			time.Sleep(30 * time.Second)
			mutex.Lock()
			output, err := json.Marshal(Blockchain)
			if err != nil {
				log.Fatal(err)
			}
			mutex.Unlock()
			io.WriteString(conn, string(output))
		}
	}()

	for _ = range bcServer {
		spew.Dump(Blockchain)
	}

}

// make sure block is valid by checking index, and comparing the hash of the previous block
func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}

	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}

// verifying chain length
func replaceChain(newBlocks []Block) {
	//mutex.Lock()
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
	//mutex.Unlock()
}

// sha256
func calculateHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// create block
func generateBlock(oldBlock Block, BPM int) (Block, error) {

	var newBlock Block

	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.BPM = BPM
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = calculateHash(newBlock)

	return newBlock, nil
}
