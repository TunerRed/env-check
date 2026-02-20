package scan

import (
    "fmt"
    "io/fs"
    "path/filepath"
    "regexp"
    "strings"
)

// Groups maps groupKey -> env -> path
type Groups map[string]map[string]string

func FindGroups(dir string, envs []string) (Groups, error) {
    // build regex
    for i := range envs {
        envs[i] = regexp.QuoteMeta(envs[i])
    }
    envPattern := strings.Join(envs, "|")
    envRe := regexp.MustCompile(fmt.Sprintf(`^(.+)-(%s)\.(.+)$`, envPattern))

    groups := make(Groups)
    err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
        if err != nil || d.IsDir() {
            return err
        }
        name := filepath.Base(path)
        m := envRe.FindStringSubmatch(name)
        if len(m) == 4 {
            // group by same directory: use path relative to root dir if possible
            dirPath := filepath.Dir(path)
            rel, rerr := filepath.Rel(dir, dirPath)
            var groupKey string
            fileBase := m[1] + "." + m[3]
            if rerr == nil && rel != "." {
                groupKey = filepath.ToSlash(filepath.Join(rel, fileBase))
            } else {
                groupKey = fileBase
            }
            env := m[2]
            if _, ok := groups[groupKey]; !ok {
                groups[groupKey] = make(map[string]string)
            }
            groups[groupKey][env] = path
        }
        return nil
    })
    if err != nil {
        return nil, err
    }
    return groups, nil
}
