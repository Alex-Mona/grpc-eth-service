package main

import (
    "context"
    "crypto/ecdsa"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "log"

    pb "grpc-eth-service/internal/proto"
    "github.com/ethereum/go-ethereum/crypto"
    "google.golang.org/grpc"
)

func main() {
    // Connect to gRPC server
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewAccountServiceClient(conn)

    // Generate a new Ethereum wallet
    privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
    if err != nil {
        log.Fatalf("failed to generate key: %v", err)
    }

    address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()

    // Create a message to sign
    message := []byte("some message")

    // Hash the message
    hash := sha256.Sum256(message)

    // Sign the hash
    signature, err := crypto.Sign(hash[:], privateKey)
    if err != nil {
        log.Fatalf("failed to sign message: %v", err)
    }

    // Encode the signature to Base64 for transmission
    signatureBase64 := base64.StdEncoding.EncodeToString(signature)

    // Call GetAccount method
    resp, err := client.GetAccount(context.Background(), &pb.GetAccountRequest{
        EthereumAddress: address,
        CryptoSignature: signatureBase64, // We transmit the encrypted signature
    })
    if err != nil {
        log.Fatalf("error calling GetAccount: %v", err)
    }

    log.Printf("Gastoken Balance: %s, Wallet Nonce: %d", resp.GetGastokenBalance(), resp.GetWalletNonce())
}
