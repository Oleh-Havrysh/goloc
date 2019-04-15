package goloc

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"
)

func localizationsCount(localizations Localizations) map[Lang]int {
	result := map[Lang]int{}
	for _, keyLoc := range localizations {
		for lang := range keyLoc {
			result[lang]++
		}
	}
	return result
}

func newWriter(
	platform Platform,
	dir ResDir,
	lang Lang,
	defLocLang Lang,
	defLocPath string,
) (file *os.File, writer *bufio.Writer, error error) {
	// Get actual resource file dir and name
	resDir, fileName, err := localizationFilePath(platform, dir, lang, defLocLang, defLocPath)
	if err != nil {
		error = err
		return
	}

	// Create all intermediate directories
	err = os.MkdirAll(resDir, os.ModePerm)
	if err != nil {
		error = err
		return
	}

	// Create actual localization file
	file, err = os.Create(filepath.Join(resDir, fileName))
	if err != nil {
		error = err
		return
	}

	// Create a new writer for the localization file
	writer = bufio.NewWriter(file)

	return
}

func writeHeaders(platform Platform, buffers map[Lang]*bytes.Buffer, t time.Time) error {
	headerArgs := &HeaderArgs{}
	for lang, buf := range buffers {
		headerArgs.Lang = lang
		headerArgs.Time = t
		if _, err := buf.WriteString(platform.Header(headerArgs)); err != nil {
			return err
		}
	}
	return nil
}

func writeFooters(platform Platform, buffers map[Lang]*bytes.Buffer) error {
	footerArgs := &FooterArgs{}
	for lang, buf := range buffers {
		footerArgs.Lang = lang
		if _, err := buf.WriteString(platform.Footer(footerArgs)); err != nil {
			return err
		}
	}
	return nil
}

func writeBuffers(
	platform Platform,
	dir ResDir,
	localizations Localizations,
	defLocLang Lang,
	defLocPath string,
	buffers map[Lang]*bytes.Buffer,
) error {
	ch := make(chan error, len(buffers))
	for lang, buf := range buffers {
		go func(lang Lang, buf *bytes.Buffer) {
			file, writer, err := newWriter(platform, dir, lang, defLocLang, defLocPath)
			if err != nil {
				ch <- err
				return
			}
			defer file.Close()
			defer writer.Flush()

			if _, err = writer.WriteString(buf.String()); err != nil {
				ch <- err
				return
			}
			ch <- nil
		}(lang, buf)
	}

	for _ = range buffers {
		if err := <-ch; err != nil {
			return err
		}
	}

	return nil
}

// WriteLocalizations writes localization files into platform-defined directories.
func WriteLocalizations(
	platform Platform,
	dir ResDir,
	localizations Localizations,
	defLocLang Lang,
	defLocPath string,
) (error error) {
	// Make sure we can access resources dir
	if _, error = os.Stat(dir); error != nil {
		return
	}

	locIndices := map[Lang]int{}
	locCounts := localizationsCount(localizations)
	locStringArgs := &LocalizedStringArgs{}

	// Prepare string buffers for each language
	buffers := map[Lang]*bytes.Buffer{}
	for lang := range locCounts {
		buffers[lang] = bytes.NewBufferString("")
	}

	// Write headers
	if error = writeHeaders(platform, buffers, time.Now()); error != nil {
		return
	}

	// Write localization strings
	for key, keyLoc := range localizations {
		for lang, value := range keyLoc {
			buf := buffers[lang]

			// Update arguments
			locStringArgs.Index = locIndices[lang]
			locStringArgs.IsLast = locIndices[lang]+1 >= locCounts[lang]
			locStringArgs.Key = key
			locStringArgs.Lang = lang
			locStringArgs.Value = value

			// Write a localized string
			localizedString := platform.LocalizedString(locStringArgs)
			if _, error = buf.WriteString(localizedString); error != nil {
				return
			}
			locIndices[lang]++
		}
	}

	// Write footers
	if error = writeFooters(platform, buffers); error != nil {
		return
	}

	// Write all buffers to files
	if error = writeBuffers(platform, dir, localizations, defLocLang, defLocPath, buffers); error != nil {
		return
	}

	return nil
}

func localizationFilePath(platform Platform, dir ResDir, lang Lang, defLocLang Lang, defLocPath string) (resDir string, fileName string, err error) {
	// Handle default language
	if len(defLocLang) > 0 && lang == defLocLang && len(defLocPath) > 0 {
		resDir = path.Dir(defLocPath)
		fileName = path.Base(defLocPath)
	} else {
		filePath := platform.LocalizationFilePath(lang, dir)
		if len(filePath) == 0 {
			return "", "", &emptyLocalizationFilePath{}
		}

		resDir = path.Dir(filePath)
		fileName = path.Base(filePath)
	}
	return
}

// region Errors

type emptyLocalizationFilePath struct {
}

func (e *emptyLocalizationFilePath) Error() string {
	return fmt.Sprintf("empty localization file path")
}

// endregion
