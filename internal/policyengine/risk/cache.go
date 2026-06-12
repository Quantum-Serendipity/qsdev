package risk

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

type CacheManager struct {
	baseDir string
	ttl     time.Duration
}

func NewCacheManager(baseDir string, ttl time.Duration) *CacheManager {
	return &CacheManager{
		baseDir: baseDir,
		ttl:     ttl,
	}
}

func (c *CacheManager) Get(ecosystem, name, version string) (*PackageScore, error) {
	path := c.cachePath(ecosystem, name, version)

	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("checking cache file: %w", err)
	}

	if time.Since(fi.ModTime()) > c.ttl {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading cache file: %w", err)
	}

	var score PackageScore
	if err := json.Unmarshal(data, &score); err != nil {
		return nil, fmt.Errorf("decoding cached score: %w", err)
	}

	return &score, nil
}

func (c *CacheManager) Put(score *PackageScore) error {
	if err := os.MkdirAll(c.baseDir, fileutil.ModeDirDefault); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	data, err := json.Marshal(score)
	if err != nil {
		return fmt.Errorf("encoding score: %w", err)
	}

	path := c.cachePath(string(score.Ecosystem), score.PackageName, score.PackageVersion)
	if err := os.WriteFile(path, data, fileutil.ModeReadWrite); err != nil {
		return fmt.Errorf("writing cache file: %w", err)
	}

	return nil
}

func (c *CacheManager) cachePath(ecosystem, name, version string) string {
	key := fmt.Sprintf("%s/%s@%s", ecosystem, name, version)
	hash := sha256.Sum256([]byte(key))
	filename := fmt.Sprintf("%x.json", hash[:8])
	return filepath.Join(c.baseDir, filename)
}
