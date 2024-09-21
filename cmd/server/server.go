package main

import (
    "context"
    "crypto/ecdsa"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "log"
    "net"

    "github.com/ethereum/go-ethereum/accounts/abi/bind"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/ethclient"
    pb "grpc-eth-service/internal/proto"
    "google.golang.org/grpc"
    "io"
    erc20 "grpc-eth-service/pkg/erc20" // Path to the generated ERC-20 ABI file
)

// Server implements the gRPC methods
type Server struct {
    pb.UnimplementedAccountServiceServer
}

// GenerateECDSAKeys generates a new ECDSA private and public key pair
func GenerateECDSAKeys() (*ecdsa.PrivateKey, error) {
    // Generating a private key
    privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
    if err != nil {
        return nil, err
    }
    return privateKey, nil
}

// SignMessage signs a given message using the ECDSA private key
func SignMessage(privateKey *ecdsa.PrivateKey, msg []byte) ([]byte, error) {
    hash := sha256.Sum256(msg)
    signature, err := crypto.Sign(hash[:], privateKey)
    if err != nil {
        return nil, err
    }
    return signature, nil
}

// VerifySignature verifies an Ethereum signature
func VerifySignature(pubKey *ecdsa.PublicKey, msg []byte, signature []byte) bool {
    hash := sha256.Sum256(msg)
    recoveredPubKey, err := crypto.SigToPub(hash[:], signature)
    if err != nil {
        return false
    }
    return recoveredPubKey.Equal(pubKey)
}

// GetAccount method implementation
func (s *Server) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
    ethereumAddress := common.HexToAddress(req.GetEthereumAddress())
    cryptoSignature := req.GetCryptoSignature()

    // Decoding the signature from Base64
    signature, err := base64.StdEncoding.DecodeString(cryptoSignature)
    if err != nil {
        return nil, fmt.Errorf("failed to decode signature: %v", err)
    }

    // Validate the crypto signature
    if !isValidSignature(ethereumAddress, signature) {
        return nil, fmt.Errorf("invalid signature")
    }

    // Example values for gastoken_balance and wallet_nonce
    gasTokenBalance := "100" // Replace this with actual logic for balance
    walletNonce := uint64(1) // Replace with actual nonce retrieval logic

    return &pb.GetAccountResponse{
        GastokenBalance: gasTokenBalance,
        WalletNonce:     walletNonce,
    }, nil
}

// GetAccounts method implementation (streaming)
func (s *Server) GetAccounts(stream pb.AccountService_GetAccountsServer) error {
    for {
        req, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }

        // Getting the balance for each Ethereum address in the request
        for _, ethAddress := range req.GetEthereumAddresses() {
            balance := getERC20Balance(ethAddress, req.GetErc20TokenAddress())

            // Stream the response back to the client
            response := &pb.GetAccountsResponse{
                EthereumAddress: ethAddress,
                Erc20Balance:    balance,
            }
            if err := stream.Send(response); err != nil {
                return err
            }
        }
    }
    return nil
}

// getERC20Balance retrieves the balance of an ERC-20 token for a given address
func getERC20Balance(address, tokenAddress string) string {
    // Connect to Ethereum node (you can use public nodes such as Infura)
    client, err := ethclient.Dial("https://mainnet.infura.io/v3/0e7d2c4248f2435da085101327eaa5e3")
    if err != nil {
        log.Fatalf("Failed to connect to the Ethereum client: %v", err)
    }

    // Convert addresses to the common.Address type
    tokenAddressCommon := common.HexToAddress(tokenAddress)
    userAddressCommon := common.HexToAddress(address)

    // Obtaining a copy of the ERC-20 contract
    contract, err := erc20.NewErc20(tokenAddressCommon, client)
    if err != nil {
        log.Fatalf("Failed to load ERC20 contract: %v", err)
    }

    // Calling the `balanceOf` function of an ERC-20 contract
    balance, err := contract.BalanceOf(&bind.CallOpts{}, userAddressCommon)
    if err != nil {
        log.Fatalf("Failed to retrieve token balance: %v", err)
    }

    // Convert balance from big.Int format to string for convenient display
    return balance.String() // Returning balance in string format
}

// Helper function to validate signature (updated)
func isValidSignature(address common.Address, signature []byte) bool {
    // For demo purposes, simulate the process of signature verification
    msg := []byte("some message")
    hash := sha256.Sum256(msg)

    // Recover the public key from the signature
    pubKey, err := crypto.SigToPub(hash[:], signature)
    if err != nil {
        return false
    }

    // Validate that the recovered public key corresponds to the provided address
    return crypto.PubkeyToAddress(*pubKey) == address
}

func main() {
    // Generate keys and sign a message
    privateKey, err := GenerateECDSAKeys()
    if err != nil {
        log.Fatalf("Error generating keys: %v", err)
    }

    publicKey := &privateKey.PublicKey
    msg := []byte("test message")
    signature, err := SignMessage(privateKey, msg)
    if err != nil {
        log.Fatalf("Error signing message: %v", err)
    }

    // Verify the signature
    isValid := VerifySignature(publicKey, msg, signature)
    fmt.Printf("Signature valid: %v\n", isValid)

    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    grpcServer := grpc.NewServer()
    pb.RegisterAccountServiceServer(grpcServer, &Server{})

    log.Printf("server listening at %v", lis.Addr())
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
