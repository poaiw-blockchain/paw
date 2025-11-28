$content = Get-Content x/compute/module.go -Raw
$pattern = 'func \(AppModuleBasic\) GetQueryCmd\(\) \*cobra.Command \{[\s\S]*?\n\}'
$new = @"func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   types.ModuleName,
		Short: \"Query commands for the compute module\",
	}

	cmd.AddCommand(
		queryParamsCmd(),
		queryProviderCmd(),
		queryProvidersCmd(),
		queryRequestCmd(),
		queryRequestsCmd(),
	)

	return cmd
}
"@
$content = [regex]::Replace($content, $pattern, $new)
Set-Content x/compute/module.go -Value $content
