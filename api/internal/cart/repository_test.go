package cart

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestCartGet(t *testing.T) {
	t.Run("stores cart items in redis hash keyed by owner", func(t *testing.T) {
		db := newTestRedisClient(t)
		repo := CreateNewCartRepository(db)

		ownerID := uuid.New()
		firstProductID := uuid.New()
		secondProductID := uuid.New()

		cart := &Cart{
			Owner: ownerID,
			Items: map[uuid.UUID]CartItem{
				firstProductID: {
					ProductID: firstProductID,
					Qty:       2,
					Price:     15000,
				},
				secondProductID: {
					ProductID: secondProductID,
					Qty:       1,
					Price:     5000,
				},
			},
		}

		ctx := context.Background()
		if err := repo.Save(ctx, cart); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		result, err := repo.Get(ctx, ownerID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		resultID := result.Items[firstProductID].ProductID
		expectedID := cart.Items[firstProductID].ProductID

		if resultID != expectedID {
			t.Fatalf("expected stored item %+v, got %+v", expectedID, resultID)
		}

	})

}

func TestCartRepositorySave(t *testing.T) {
	t.Run("stores cart items in redis hash keyed by owner", func(t *testing.T) {
		db := newTestRedisClient(t)
		repo := CreateNewCartRepository(db)

		ownerID := uuid.New()
		firstProductID := uuid.New()
		secondProductID := uuid.New()

		cart := &Cart{
			Owner: ownerID,
			Items: map[uuid.UUID]CartItem{
				firstProductID: {
					ProductID: firstProductID,
					Qty:       2,
					Price:     15000,
				},
				secondProductID: {
					ProductID: secondProductID,
					Qty:       1,
					Price:     5000,
				},
			},
		}

		ctx := context.Background()
		if err := repo.Save(ctx, cart); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		storedItems, err := db.HGetAll(ctx, "cart:"+ownerID.String()).Result()
		if err != nil {
			t.Fatalf("expected redis read to succeed, got %v", err)
		}

		if len(storedItems) != 2 {
			t.Fatalf("expected 2 stored items, got %d", len(storedItems))
		}

		assertStoredCartItem(t, storedItems, firstProductID, cart.Items[firstProductID])
		assertStoredCartItem(t, storedItems, secondProductID, cart.Items[secondProductID])
	})

	t.Run("returns nil for empty cart items", func(t *testing.T) {
		db := newTestRedisClient(t)
		repo := CreateNewCartRepository(db)

		cart := &Cart{
			Owner: uuid.New(),
			Items: map[uuid.UUID]CartItem{},
		}

		if err := repo.Save(context.Background(), cart); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		exists, err := db.Exists(context.Background(), "cart:"+cart.Owner.String()).Result()
		if err != nil {
			t.Fatalf("expected redis exists check to succeed, got %v", err)
		}

		if exists != 0 {
			t.Fatalf("expected no redis key to be created, got exists=%d", exists)
		}
	})
}

func assertStoredCartItem(t *testing.T, storedItems map[string]string, productID uuid.UUID, expected CartItem) {
	t.Helper()

	rawItem, ok := storedItems[productID.String()]
	if !ok {
		t.Fatalf("expected product %s to be stored", productID)
	}

	var storedItem CartItem
	if err := json.Unmarshal([]byte(rawItem), &storedItem); err != nil {
		t.Fatalf("expected stored item to be valid json, got %v", err)
	}

	if storedItem != expected {
		t.Fatalf("expected stored item %+v, got %+v", expected, storedItem)
	}
}

func newTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	server := startRedisServer(t)
	t.Cleanup(func() {
		if err := server.stop(); err != nil {
			t.Fatalf("failed to stop redis server: %v", err)
		}
	})

	db := redis.NewClient(&redis.Options{Addr: server.addr})
	ctx := context.Background()

	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := db.Ping(ctx).Err(); err == nil {
			break
		}

		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for redis server to start")
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close redis client: %v", err)
		}
	})

	return db
}

type testRedisServer struct {
	addr string
	cmd  *exec.Cmd
}

func startRedisServer(t *testing.T) *testRedisServer {
	t.Helper()

	redisPath, err := exec.LookPath("redis-server")
	if err != nil {
		t.Skip("redis-server is required to run cart repository tests")
	}

	port := reserveRedisPort(t)
	dataDir := t.TempDir()
	logPath := filepath.Join(dataDir, "redis.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("failed to create redis log file: %v", err)
	}

	cmd := exec.Command(
		redisPath,
		"--bind", "127.0.0.1",
		"--port", strconv.Itoa(port),
		"--save", "",
		"--appendonly", "no",
		"--dir", dataDir,
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		t.Fatalf("failed to start redis-server: %v", err)
	}

	t.Cleanup(func() {
		_ = logFile.Close()
	})

	return &testRedisServer{
		addr: fmt.Sprintf("127.0.0.1:%d", port),
		cmd:  cmd,
	}
}

func (s *testRedisServer) stop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}

	err := s.cmd.Process.Kill()
	waitErr := s.cmd.Wait()
	if err != nil && err.Error() != "os: process already finished" {
		return err
	}
	if waitErr != nil {
		_, ok := waitErr.(*exec.ExitError)
		if !ok {
			return waitErr
		}
	}

	return nil
}

func reserveRedisPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve redis port: %v", err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("failed to get reserved tcp address")
	}

	return addr.Port
}
