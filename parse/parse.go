package parse

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"

    "env-check/rules"

    "github.com/fatih/color"
    "github.com/magiconair/properties"
    toml "github.com/pelletier/go-toml"
    "gopkg.in/yaml.v3"
)

var ipRe = regexpMustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)

func regexpMustCompile(s string) *regexp.Regexp {
    return regexp.MustCompile(s)
}

type Result struct {
    Criticals []string
    Warnings  []string
}

// CheckGroups performs parsing, structural comparison and ip checks
func CheckGroups(groups map[string]map[string]string, baseline string, criticalEnvs map[string]struct{}, ruleMap rules.RuleSet) *Result {
    res := &Result{}
    for grp, envFiles := range groups {
        fmt.Printf("\nChecking group: %s\n", grp)
        basePath, hasBase := envFiles[baseline]
        baseKeys := map[string]string{}
        if hasBase {
            _, baseKeys = parseAndFlatten(basePath)
        }
        for env, path := range envFiles {
            fmt.Printf(" - %s: %s\n", env, path)
            keys, vals := parseAndFlatten(path)
            if hasBase {
                // if baseline is uat and env is prod or if env is in criticalEnvs, check missing keys
                if baseline == "uat" && env == "prod" {
                    for k := range baseKeys {
                        if _, ok := keys[k]; !ok {
                            msg := fmt.Sprintf("CRITICAL: group=%s prod missing key '%s' (baseline=%s)", grp, k, baseline)
                            res.Criticals = append(res.Criticals, msg)
                            color.New(color.FgRed).Printf("  %s\n", msg)
                        }
                    }
                }
            }
            ips := extractIPsFromVals(vals)
            for ip := range ips {
                if _, isCritical := criticalEnvs[env]; isCritical {
                    for other := range ruleMap {
                        if other == env {
                            continue
                        }
                        if containsIP(ruleMap[other], ip) {
                            msg := fmt.Sprintf("CRITICAL: %s contains IP %s from %s rules", path, ip, other)
                            res.Criticals = append(res.Criticals, msg)
                            color.New(color.FgRed).Printf("  %s\n", msg)
                        }
                    }
                }
                if !ipInAny(ruleMap, ip) {
                    msg := fmt.Sprintf("WARNING: %s contains unknown IP %s", path, ip)
                    res.Warnings = append(res.Warnings, msg)
                    color.New(color.FgYellow).Printf("  %s\n", msg)
                }
            }
        }
    }
    return res
}

func containsIP(set map[string]struct{}, ip string) bool {
    _, ok := set[ip]
    return ok
}

func ipInAny(ruleMap rules.RuleSet, ip string) bool {
    for _, s := range ruleMap {
        if containsIP(s, ip) {
            return true
        }
    }
    return false
}

func extractIPsFromVals(vals map[string]string) map[string]struct{} {
    out := make(map[string]struct{})
    for _, v := range vals {
        for _, m := range ipRe.FindAllString(v, -1) {
            out[m] = struct{}{}
        }
    }
    return out
}

func parseAndFlatten(path string) (map[string]struct{}, map[string]string) {
    ext := strings.ToLower(filepath.Ext(path))
    ext = strings.TrimPrefix(ext, ".")
    switch ext {
    case "yml", "yaml":
        m := map[string]interface{}{}
        data, _ := os.ReadFile(path)
        _ = yaml.Unmarshal(data, &m)
        flat := map[string]string{}
        flattenMapInterface(m, "", flat)
        keys := make(map[string]struct{})
        for k := range flat {
            keys[k] = struct{}{}
        }
        return keys, flat
    case "toml":
        tree, err := toml.LoadFile(path)
        flat := map[string]string{}
        if err == nil {
            toMap := tree.ToMap()
            flattenMapInterface(toMap, "", flat)
        }
        keys := make(map[string]struct{})
        for k := range flat {
            keys[k] = struct{}{}
        }
        return keys, flat
    case "properties":
        p, err := properties.LoadFile(path, properties.UTF8)
        flat := map[string]string{}
        if err == nil {
            for _, k := range p.Keys() {
                val, _ := p.Get(k)
                flat[k] = val
            }
        }
        keys := make(map[string]struct{})
        for k := range flat {
            keys[k] = struct{}{}
        }
        return keys, flat
    case "json":
        data, _ := os.ReadFile(path)
        var m map[string]interface{}
        _ = json.Unmarshal(data, &m)
        flat := map[string]string{}
        flattenMapInterface(m, "", flat)
        keys := make(map[string]struct{})
        for k := range flat {
            keys[k] = struct{}{}
        }
        return keys, flat
    default:
        flat := map[string]string{}
        f, err := os.Open(path)
        if err == nil {
            scanner := bufio.NewScanner(f)
            for scanner.Scan() {
                line := strings.TrimSpace(scanner.Text())
                if line == "" || strings.HasPrefix(line, "#") {
                    continue
                }
                if idx := strings.Index(line, "="); idx > 0 {
                    k := strings.TrimSpace(line[:idx])
                    v := strings.TrimSpace(line[idx+1:])
                    flat[k] = v
                }
            }
            f.Close()
        }
        keys := make(map[string]struct{})
        for k := range flat {
            keys[k] = struct{}{}
        }
        return keys, flat
    }
}

func flattenMapInterface(m interface{}, prefix string, out map[string]string) {
    switch t := m.(type) {
    case map[string]interface{}:
        for k, v := range t {
            key := k
            if prefix != "" {
                key = prefix + "." + k
            }
            flattenMapInterface(v, key, out)
        }
    case map[interface{}]interface{}:
        for kk, vv := range t {
            ks := fmt.Sprintf("%v", kk)
            key := ks
            if prefix != "" {
                key = prefix + "." + ks
            }
            flattenMapInterface(vv, key, out)
        }
    case []interface{}:
        for i, v := range t {
            key := fmt.Sprintf("%s[%d]", prefix, i)
            flattenMapInterface(v, key, out)
        }
    default:
        out[prefix] = fmt.Sprintf("%v", t)
    }
}
