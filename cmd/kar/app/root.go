/*
Copyright © 2024

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"context"
	stdErrors "errors"

	runner "github.com/electrocucaracha/kubevirt-actions-runner/internal"
	"github.com/electrocucaracha/kubevirt-actions-runner/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// CleanupContextFunc creates the context used to delete runner resources.
type CleanupContextFunc func(context.Context) (context.Context, context.CancelFunc)

func NewRootCommand(ctx context.Context, runner runner.Runner, opts Opts, cleanupContext CleanupContextFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kar",
		Short: "Tool that creates a GitHub Self-Host runner with Kubevirt Virtual Machine Instance",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return initializeConfig(cmd)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return run(ctx, runner, opts, cleanupContext)
		},
	}

	installFlags(cmd.Flags(), &opts)

	return cmd
}

func run(ctx context.Context, runner runner.Runner, opts Opts, cleanupContext CleanupContextFunc) error {
	log := utils.GetLogger()

	err := runner.CreateResources(ctx, opts.VMTemplate, opts.RunnerName, opts.JitConfig)
	if err != nil {
		return errors.Wrap(err, "fail to create resources")
	}

	log.Println("Virtual Machine runner resources created successfully")

	err = runner.WaitForVirtualMachineInstance(ctx)
	if err != nil {
		if !stdErrors.Is(errors.Cause(err), context.Canceled) {
			cleanupErr := deleteResources(ctx, runner, cleanupContext)
			if cleanupErr != nil {
				log.Warnf("failed to delete resources after wait failure: %v", cleanupErr)
			}
		}

		return errors.Wrap(err, "fail to wait for resources")
	}

	log.Println("Virtual Machine runner completed successfully")

	err = deleteResources(ctx, runner, cleanupContext)
	if err != nil {
		return errors.Wrap(err, "fail to delete resources")
	}

	log.Println("Virtual Machine runner deleted successfully")

	return nil
}

func deleteResources(ctx context.Context, runner runner.Runner, cleanupContext CleanupContextFunc) error {
	cleanupCtx, cancel := cleanupContext(ctx)
	defer cancel()

	return runner.DeleteResources(cleanupCtx)
}
