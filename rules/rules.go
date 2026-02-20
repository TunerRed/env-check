package rules

import (
    "bufio"
    "os"
    "path/filepath"
    "strings"
)

type RuleSet map[string]map[string]struct{}

func LoadRules(dir string) (RuleSet, error) {
    res := make(RuleSet)
    entries, err := os.ReadDir(dir)
    if err != nil {
        return nil, err
    }
    for _, e := range entries {
        if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
            continue
        }
        name := strings.TrimSuffix(e.Name(), ".txt")
        data, err := os.ReadFile(filepath.Join(dir, e.Name()))
        if err != nil {
            return nil, err
        }
        set := make(map[string]struct{})
        scanner := bufio.NewScanner(strings.NewReader(string(data)))
        for scanner.Scan() {
            line := strings.TrimSpace(scanner.Text())
            if line == "" || strings.HasPrefix(line, "#") {
                continue
            }
            set[line] = struct{}{}
        }
        res[name] = set
    }
    return res, nil
}

func DeriveEnvs(rs RuleSet) []string {
    out := []string{}
    for k := range rs {
        out = append(out, k)
    }
    if len(out) == 0 {
        return []string{"test", "uat", "prod"}
    }
    return out
}

func DetermineBaseline(envs []string, baseline string) string {
    if baseline != "" {
        return baseline
    }
    for _, e := range envs {
        if e == "uat" {
            return "uat"
        }
    }
    return envs[0]
}

func DeriveCriticalEnvs(envs []string, criticalFlag string) map[string]struct{} {
    out := map[string]struct{}{}
    if criticalFlag != "" {
        for _, s := range strings.Split(criticalFlag, ",") {
            ss := strings.TrimSpace(s)
            if ss != "" {
                out[ss] = struct{}{}
            }
        }
        return out
    }
    for _, e := range envs {
        if e == "prod" {
            out["prod"] = struct{}{}
            break
        }
    }
    return out
}
