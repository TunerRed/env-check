package cmd

import (
    "flag"
    "fmt"
    "os"

    "env-check/parse"
    "env-check/rules"
    "env-check/scan"

    "github.com/fatih/color"
)

func Run() {
    dir := flag.String("dir", ".", "target directory to scan")
    envsDir := flag.String("envs", "envs", "envs directory (uat.txt etc)")
    baseline := flag.String("baseline", "", "baseline env for structural comparison")
    criticalFlag := flag.String("critical-env", "", "comma-separated envs that trigger cross-env critical checks (default: prod if exists)")
    flag.Parse()

    ruleMap, err := rules.LoadRules(*envsDir)
    if err != nil {
        fmt.Println("Failed to load rules:", err)
        os.Exit(-1)
    }

    envs := rules.DeriveEnvs(ruleMap)
    *baseline = rules.DetermineBaseline(envs, *baseline)
    criticalEnvs := rules.DeriveCriticalEnvs(envs, *criticalFlag)

    groups, err := scan.FindGroups(*dir, envs)
    if err != nil {
        fmt.Println("Scan error:", err)
        os.Exit(-1)
    }

    // validate that each group contains all envs defined by rules
    // if some env files are missing for a group, treat as Critical
    missingRes := &parse.Result{}
    for grp, envFiles := range groups {
        for _, e := range envs {
            if _, ok := envFiles[e]; !ok {
                msg := fmt.Sprintf("CRITICAL: group=%s missing file for env '%s'", grp, e)
                missingRes.Criticals = append(missingRes.Criticals, msg)
                color.New(color.FgRed).Printf("  %s\n", msg)
            }
        }
    }

    res := parse.CheckGroups(groups, *baseline, criticalEnvs, ruleMap)
    // merge missingRes into res
    res.Criticals = append(missingRes.Criticals, res.Criticals...)
    res.Warnings = append(missingRes.Warnings, res.Warnings...)

    if len(res.Criticals) > 0 {
        color.New(color.FgRed).Printf("Found %d critical issues\n", len(res.Criticals))
        os.Exit(2)
    }
    if len(res.Warnings) > 0 {
        color.New(color.FgYellow).Printf("Found %d warnings\n", len(res.Warnings))
        os.Exit(1)
    }
    color.New(color.FgGreen).Printf("No issues found\n")
    os.Exit(0)
}
