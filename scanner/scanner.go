package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"

	"github.com/varkuru/creddrift/config"
)

// Match represents a found potential secret.
type Match struct {
	File     string
	LineNum  int
	MatchTxt string
	Type     string
	Entropy  float64
}

// SecretScanner handles the file traversal and matching.
type SecretScanner struct {
	config   *config.Config
	awsRegex *regexp.Regexp
	ghRegex  *regexp.Regexp
	genRegex *regexp.Regexp
}

// NewScanner initializes a SecretScanner using the provided config.
func NewScanner(cfg *config.Config) *SecretScanner {
	return &SecretScanner{
		config:   cfg,
		awsRegex: regexp.MustCompile(`\bAKIA[A-Z0-9]{16}\b`),
		ghRegex:  regexp.MustCompile(`\bghp_[a-zA-Z0-9]{36}\b`),
		genRegex: regexp.MustCompile(`(?i)(api_key|password)\s*[:=]\s*["']?([^"'\s]+)["']?`),
	}
}

// Scan executes the directory walk and initiates file scanning.
func (s *SecretScanner) Scan() ([]Match, error) {
	var results []Match

	for _, target := range s.config.ScanTargets {
		err := filepath.WalkDir(target, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Ignore configured directories
			if d.IsDir() {
				// Don't skip the base target if it happens to be named like an ignored directory
				if path != target {
					for _, ignored := range s.config.IgnoreDirs {
						if d.Name() == ignored {
							return filepath.SkipDir
						}
					}
				}
				return nil
			}

			// Check and skip binary files
			isBinary, err := isBinaryFile(path)
			if err != nil || isBinary {
				return nil
			}

			matches, _ := s.scanFile(path)
			results = append(results, matches...)
			return nil
		})
		if err != nil {
			return results, err
		}
	}
	return results, nil
}

// scanFile reads a text file line by line and looks for patterns.
func (s *SecretScanner) scanFile(path string) ([]Match, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []Match
	lineNum := 1
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if awsm := s.awsRegex.FindString(line); awsm != "" {
			matches = append(matches, Match{File: path, LineNum: lineNum, MatchTxt: awsm, Type: "AWS", Entropy: CalculateShannonEntropy(awsm)})
		}

		if ghm := s.ghRegex.FindString(line); ghm != "" {
			matches = append(matches, Match{File: path, LineNum: lineNum, MatchTxt: ghm, Type: "GitHub", Entropy: CalculateShannonEntropy(ghm)})
		}

		// Use FindAllStringSubmatch to catch generic keys mapped to strings
		for _, submatches := range s.genRegex.FindAllStringSubmatch(line, -1) {
			if len(submatches) >= 3 {
				secretVal := submatches[2]
				entropy := CalculateShannonEntropy(secretVal)
				matchType := "Generic"
				if entropy <= 3.5 {
					matchType = "Weak_Secret"
				}
				matches = append(matches, Match{File: path, LineNum: lineNum, MatchTxt: secretVal, Type: matchType, Entropy: entropy})
			}
		}
		lineNum++
	}
	return matches, scanner.Err()
}

// isBinaryFile reads the first 512 bytes and checks for null bytes.
func isBinaryFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err.Error() != "EOF" {
		return false, err
	}
	
	// If it contains a null byte, consider it binary.
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true, nil
		}
	}
	return false, nil
}
