package keystore

import (
  "io/ioutil"
  "os"
  "path/filepath"
  "strings"
  "sync"
  "time"

  set "gopkg.in/fatih/set.v0"
)

type fileCache struct {
  all     *set.SetNonTS // Non-thread safe set data structure
  lastMod time.Time
  mu      sync.RWMutex
}

// scan performs a new scan on the given directory, compares against the already
// cached filenames, and returns file sets: creates, deletes, updates.
func (fc *fileCache) scan(keyDir string) (set.Interface, set.Interface, set.Interface, error) {
  // List all the failes from the keystore folder
  files, err := ioutil.ReadDir(keyDir)
  if err != nil {
    return nil, nil, nil, err
  }

  fc.mu.Lock()
  defer fc.mu.Unlock()

  // Iterate all the files and gather their metadata
  all := set.NewNonTS()
  mods := set.NewNonTS()

  var newLastMod time.Time
  for _, fi := range files {
    // Skip any non-key files from the folder
    path := filepath.Join(keyDir, fi.Name())
    if skipKeyFile(fi) {
      continue
    }
    // Gather the set of all and fresly modified files
    all.Add(path)

    modified := fi.ModTime()
    if modified.After(fc.lastMod) {
      mods.Add(path)
    }
    if modified.After(newLastMod) {
      newLastMod = modified
    }
  }

  // Update the tracked files and return the three sets
  deletes := set.Difference(fc.all, all)   // Deletes = previous - current
  creates := set.Difference(all, fc.all)   // Creates = current - previous
  updates := set.Difference(mods, creates) // Updates = modified - creates

  fc.all, fc.lastMod = all, newLastMod

  // Report on the scanning stats and return
  return creates, deletes, updates, nil
}

// skipKeyFile ignores editor backups, hidden files and folders/symlinks.
func skipKeyFile(fi os.FileInfo) bool {
  // Skip editor backups and UNIX-style hidden files.
  if strings.HasSuffix(fi.Name(), "~") || strings.HasPrefix(fi.Name(), ".") {
    return true
  }
  // Skip misc special files, directories (yes, symlinks too).
  if fi.IsDir() || fi.Mode()&os.ModeType != 0 {
    return true
  }
  return false
}
