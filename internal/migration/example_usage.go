package migration

import (
	"fmt"
	"log"
)

// ExampleBasicMigration demonstrates basic migration usage
func ExampleBasicMigration() {
	// Create migrator
	migrator := NewMigrator()

	// Perform complete migration
	result := migrator.Migrate()

	// Check results
	if len(result.Errors) > 0 {
		log.Printf("Migration completed with errors:")
		for _, err := range result.Errors {
			log.Printf("  - %v", err)
		}
	}

	if result.SettingsMigrated {
		log.Println("✓ Settings migrated successfully")
	}

	if result.QueueMigrated {
		log.Println("✓ Queue migrated successfully")
	}

	if result.HistoryMigrated {
		log.Println("✓ History migrated successfully")
	}

	log.Printf("Backup created at: %s", result.BackupPath)
}

// ExampleCheckMigrationNeeded demonstrates checking if migration is needed
func ExampleCheckMigrationNeeded() {
	needed, err := CheckMigrationNeeded()
	if err != nil {
		log.Fatal(err)
	}

	if needed {
		fmt.Println("Python installation detected. Migration recommended.")
		
		// Get more info
		info, err := GetMigrationInfo()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Python directory: %s\n", info["python_dir"])
		fmt.Printf("Has settings: %v\n", info["has_settings"])
		fmt.Printf("Has queue: %v\n", info["has_queue"])
	} else {
		fmt.Println("No migration needed.")
	}
}

// ExampleStepByStepMigration demonstrates step-by-step migration
func ExampleStepByStepMigration() {
	migrator := NewMigrator()

	// Step 1: Detect Python installation
	fmt.Println("Step 1: Detecting Python installation...")
	installation, err := migrator.DetectPythonInstallation()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Found Python installation at: %s\n", installation.DataDir)
	fmt.Printf("  - Has settings: %v\n", installation.HasSettings)
	fmt.Printf("  - Has queue: %v\n", installation.HasQueue)

	// Step 2: Create backup
	fmt.Println("\nStep 2: Creating backup...")
	if err := migrator.CreateBackup(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Backup created at: %s\n", installation.BackupPath)

	// Step 3: Migrate settings
	if installation.HasSettings {
		fmt.Println("\nStep 3: Migrating settings...")
		if err := migrator.MigrateSettings(); err != nil {
			log.Printf("✗ Settings migration failed: %v", err)
		} else {
			fmt.Println("✓ Settings migrated successfully")
		}
	}

	// Step 4: Migrate queue
	if installation.HasQueue {
		fmt.Println("\nStep 4: Migrating queue and history...")
		if err := migrator.MigrateQueue(); err != nil {
			log.Printf("✗ Queue migration failed: %v", err)
		} else {
			fmt.Println("✓ Queue and history migrated successfully")
		}
	}

	fmt.Println("\n✓ Migration complete!")
}

// ExampleDetectionOnly demonstrates detection without migration
func ExampleDetectionOnly() {
	detector := NewDetector()

	installation, err := detector.DetectPythonInstallation()
	if err != nil {
		fmt.Printf("No Python installation found: %v\n", err)
		return
	}

	fmt.Println("Python DeeMusic Installation Found:")
	fmt.Printf("  Directory: %s\n", installation.DataDir)
	fmt.Printf("  Settings: %v", installation.HasSettings)
	if installation.HasSettings {
		fmt.Printf(" (%s)", installation.SettingsPath)
	}
	fmt.Println()

	fmt.Printf("  Queue DB: %v", installation.HasQueue)
	if installation.HasQueue {
		fmt.Printf(" (%s)", installation.QueueDBPath)
	}
	fmt.Println()
}

// ExampleSettingsMigrationOnly demonstrates settings-only migration
func ExampleSettingsMigrationOnly() {
	detector := NewDetector()

	// Detect installation
	installation, err := detector.DetectPythonInstallation()
	if err != nil {
		log.Fatal(err)
	}

	if !installation.HasSettings {
		fmt.Println("No settings found to migrate")
		return
	}

	// Create backup
	if err := detector.CreateBackup(installation); err != nil {
		log.Fatal(err)
	}

	// Migrate settings only
	migrator := NewMigrator()
	migrator.installation = installation

	if err := migrator.MigrateSettings(); err != nil {
		log.Fatalf("Settings migration failed: %v", err)
	}

	fmt.Println("✓ Settings migrated successfully")
	fmt.Printf("Backup: %s\n", installation.BackupPath)
}

// ExampleQueueMigrationOnly demonstrates queue-only migration
func ExampleQueueMigrationOnly() {
	detector := NewDetector()

	// Detect installation
	installation, err := detector.DetectPythonInstallation()
	if err != nil {
		log.Fatal(err)
	}

	if !installation.HasQueue {
		fmt.Println("No queue database found to migrate")
		return
	}

	// Create backup
	if err := detector.CreateBackup(installation); err != nil {
		log.Fatal(err)
	}

	// Migrate queue only
	migrator := NewMigrator()
	migrator.installation = installation

	if err := migrator.MigrateQueue(); err != nil {
		log.Fatalf("Queue migration failed: %v", err)
	}

	fmt.Println("✓ Queue and history migrated successfully")
	fmt.Printf("Backup: %s\n", installation.BackupPath)
}

// ExampleWithErrorHandling demonstrates comprehensive error handling
func ExampleWithErrorHandling() {
	migrator := NewMigrator()

	// Detect installation
	installation, err := migrator.DetectPythonInstallation()
	if err != nil {
		fmt.Printf("Detection failed: %v\n", err)
		fmt.Println("This is normal if you don't have a Python installation.")
		return
	}

	// Create backup
	if err := migrator.CreateBackup(); err != nil {
		log.Fatalf("Backup failed: %v", err)
	}

	// Validate backup
	if err := migrator.detector.ValidateBackup(installation); err != nil {
		log.Fatalf("Backup validation failed: %v", err)
	}

	// Track success
	var settingsOK, queueOK bool

	// Migrate settings
	if installation.HasSettings {
		if err := migrator.MigrateSettings(); err != nil {
			log.Printf("Settings migration failed: %v", err)
			log.Println("Continuing with queue migration...")
		} else {
			settingsOK = true
		}
	}

	// Migrate queue
	if installation.HasQueue {
		if err := migrator.MigrateQueue(); err != nil {
			log.Printf("Queue migration failed: %v", err)
		} else {
			queueOK = true
		}
	}

	// Summary
	fmt.Println("\nMigration Summary:")
	fmt.Printf("  Settings: %v\n", settingsOK)
	fmt.Printf("  Queue: %v\n", queueOK)
	fmt.Printf("  Backup: %s\n", installation.BackupPath)

	if settingsOK || queueOK {
		fmt.Println("\n✓ Migration completed successfully!")
	} else {
		fmt.Println("\n✗ Migration failed. Check logs for details.")
		fmt.Println("Your data is safe in the backup directory.")
	}
}
