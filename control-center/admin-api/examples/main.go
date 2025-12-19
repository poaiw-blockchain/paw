package main

import (
	"fmt"
	"log"

	"github.com/paw-chain/paw/control-center/admin-api/client"
)

func main() {
	// Create admin API client
	cfg := &client.Config{
		BaseURL:  "http://localhost:11201",
		Username: "admin",
		Password: "admin123",
	}

	adminClient, err := client.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create admin client: %v", err)
	}
	defer adminClient.Logout()

	fmt.Println("=== Admin API Examples ===")

	// Example 1: Get current DEX parameters
	fmt.Println("1. Getting current DEX parameters...")
	dexParams, err := adminClient.GetModuleParams("dex")
	if err != nil {
		log.Printf("Error getting DEX params: %v", err)
	} else {
		fmt.Printf("Current DEX params: %+v\n\n", dexParams)
	}

	// Example 2: Update DEX parameters
	fmt.Println("2. Updating DEX swap fee rate...")
	err = adminClient.UpdateModuleParams("dex", map[string]interface{}{
		"swap_fee_rate": "0.0025",
	}, "Reduce fees to increase trading volume")
	if err != nil {
		log.Printf("Error updating DEX params: %v", err)
	} else {
		fmt.Println("DEX parameters updated successfully")
		fmt.Println()
	}

	// Example 3: View parameter history
	fmt.Println("3. Viewing parameter change history...")
	history, err := adminClient.GetParamHistory("dex", 10, 0)
	if err != nil {
		log.Printf("Error getting param history: %v", err)
	} else {
		fmt.Printf("Found %d parameter changes\n", len(history))
		for _, entry := range history {
			fmt.Printf("  - %s: %s changed from %v to %v by %s (reason: %s)\n",
				entry.Timestamp.Format("2006-01-02 15:04:05"),
				entry.Param,
				entry.OldValue,
				entry.NewValue,
				entry.ChangedBy,
				entry.Reason,
			)
		}
		fmt.Println()
	}

	// Example 4: Pause module for maintenance
	fmt.Println("4. Pausing DEX module for maintenance...")
	err = adminClient.PauseModule("dex", "Scheduled maintenance - upgrading trading engine", false)
	if err != nil {
		log.Printf("Error pausing module: %v", err)
	} else {
		fmt.Println("DEX module paused successfully")
		fmt.Println()
	}

	// Example 5: Check circuit breaker status
	fmt.Println("5. Checking circuit breaker status...")
	status, err := adminClient.GetCircuitBreakerStatus("dex")
	if err != nil {
		log.Printf("Error getting circuit breaker status: %v", err)
	} else {
		fmt.Printf("DEX Circuit Breaker Status:\n")
		fmt.Printf("  Paused: %v\n", status.Paused)
		if status.Paused {
			fmt.Printf("  Paused At: %s\n", status.PausedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Paused By: %s\n", status.PausedBy)
			fmt.Printf("  Reason: %s\n", status.Reason)
		}
		fmt.Println()
	}

	// Example 6: Resume module after maintenance
	fmt.Println("6. Resuming DEX module after maintenance...")
	err = adminClient.ResumeModule("dex", "Maintenance completed successfully")
	if err != nil {
		log.Printf("Error resuming module: %v", err)
	} else {
		fmt.Println("DEX module resumed successfully")
		fmt.Println()
	}

	// Example 7: Schedule network upgrade
	fmt.Println("7. Scheduling network upgrade...")
	err = adminClient.ScheduleUpgrade(
		"v2.0.0-upgrade",
		1000000,
		"Major upgrade with new features: improved DEX performance, enhanced oracle accuracy, optimized compute verification",
	)
	if err != nil {
		log.Printf("Error scheduling upgrade: %v", err)
	} else {
		fmt.Println("Network upgrade scheduled successfully")
		fmt.Println()
	}

	// Example 8: Check upgrade status
	fmt.Println("8. Checking upgrade status...")
	upgradeStatus, err := adminClient.GetUpgradeStatus("v2.0.0-upgrade")
	if err != nil {
		log.Printf("Error getting upgrade status: %v", err)
	} else {
		fmt.Printf("Upgrade Status:\n")
		fmt.Printf("  Name: %s\n", upgradeStatus.Name)
		fmt.Printf("  Height: %d\n", upgradeStatus.Height)
		fmt.Printf("  Status: %s\n", upgradeStatus.Status)
		fmt.Printf("  Scheduled At: %s\n", upgradeStatus.ScheduledAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Scheduled By: %s\n", upgradeStatus.ScheduledBy)
		fmt.Printf("  Info: %s\n", upgradeStatus.Info)
		fmt.Println()
	}

	// Example 9: Emergency operations (commented out for safety)
	fmt.Println("9. Emergency operations (demo - not executed)")
	fmt.Println("   To perform emergency pause:")
	fmt.Println("   adminClient.EmergencyPauseDEX(\"Critical vulnerability detected\", \"123456\")")
	fmt.Println("   adminClient.EmergencyResumeAll(\"Issues resolved\", \"123456\")")
	fmt.Println()

	// Example 10: Reset parameters to defaults
	fmt.Println("10. Demonstrating parameter reset (not executed)...")
	fmt.Println("    To reset module params:")
	fmt.Println("    adminClient.ResetModuleParams(\"dex\", \"Reverting to tested defaults after issues\")")
	fmt.Println()

	fmt.Println("=== Examples completed successfully ===")
}
