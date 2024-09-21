package main

import (
    "context"
    "log"
    "math/rand"
    "time"

    pb "grpc-eth-service/internal/proto"
    "google.golang.org/grpc"
)

func main() {
    // Connecting to the gRPC server
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewAccountServiceClient(conn)

    // Generation of 100, 1000, 10000 Ethereum addresses for testing
    testAddresses := generateEthereumAddresses(10000) // Let's generate the maximum set

    // Testing the method for 100, 1000 and 10000 addresses
    testGetAccountsPerformance(client, testAddresses[:100], 3)
    testGetAccountsPerformance(client, testAddresses[:1000], 4)
    testGetAccountsPerformance(client, testAddresses[:10000], 5)
}

// Performance testing function
func testGetAccountsPerformance(client pb.AccountServiceClient, addresses []string, tokenCount int) {
    start := time.Now()

    // Receiving a stream to send and receive data
    stream, err := client.GetAccounts(context.Background())
    if err != nil {
        log.Fatalf("error opening stream: %v", err)
    }

    // Generating real ERC-20 token addresses
    tokenAddresses := generateRealTokenAddresses(tokenCount)

    // We send requests for each token and all addresses
    for _, tokenAddress := range tokenAddresses {
        err = stream.Send(&pb.GetAccountsRequest{
            EthereumAddresses: addresses,
            Erc20TokenAddress: tokenAddress, // Send one token address at a time
        })
        if err != nil {
            log.Fatalf("failed to send request: %v", err)
        }
    }

    // Finish sending data
    if err := stream.CloseSend(); err != nil {
        log.Fatalf("failed to close stream: %v", err)
    }

    // Receiving responses from the server
    for {
        resp, err := stream.Recv()
        if err != nil {
            log.Printf("Stream finished: %v", err)
            break
        }
        log.Printf("Ethereum Address: %s, ERC20 Balance: %s", resp.GetEthereumAddress(), resp.GetErc20Balance())
    }

    elapsed := time.Since(start)
    log.Printf("Time elapsed for %d addresses and %d tokens: %s", len(addresses), tokenCount, elapsed)
}

// Generating test Ethereum addresses
func generateEthereumAddresses(count int) []string {
    addresses := make([]string, count)
    for i := 0; i < count; i++ {
        addresses[i] = randomHex(40) // Random hex string 40 characters long (Ethereum address)
    }
    return addresses
}

// We replace the generation of random token addresses with real addresses of ERC-20 tokens
func generateRealTokenAddresses(count int) []string {
    // Example of real addresses of popular tokens
    realTokenAddresses := []string{
        "0xdAC17F958D2ee523a2206206994597C13D831ec7", // USDT
        "0x6B175474E89094C44Da98b954EedeAC495271d0F", // DAI
        "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", // USDC
        "0x4fabb145d64652a948d72533023f6e7a623c7c53", // BUSD
        "0x0000000000085d4780B73119b644AE5ecd22b376", // TUSD
    }

    if count > len(realTokenAddresses) {
        count = len(realTokenAddresses)
    }

    return realTokenAddresses[:count] // We use real token addresses
}

// Generating a random hex string
func randomHex(n int) string {
    const letters = "0123456789abcdef"
    result := make([]byte, n)
    for i := range result {
        result[i] = letters[rand.Intn(len(letters))]
    }
    return "0x" + string(result)
}
