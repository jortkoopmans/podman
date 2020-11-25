package common

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/containers/common/pkg/config"
	"github.com/containers/podman/v2/cmd/podman/registry"
	"github.com/containers/podman/v2/libpod"
	"github.com/containers/podman/v2/libpod/define"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/containers/podman/v2/pkg/registries"
	systemdGen "github.com/containers/podman/v2/pkg/systemd/generate"
	"github.com/spf13/cobra"
)

var (
	// ChangeCmds is the list of valid Change commands to passed to the Commit call
	ChangeCmds = []string{"CMD", "ENTRYPOINT", "ENV", "EXPOSE", "LABEL", "ONBUILD", "STOPSIGNAL", "USER", "VOLUME", "WORKDIR"}
	// LogLevels supported by podman
	LogLevels = []string{"debug", "info", "warn", "error", "fatal", "panic"}
)

type completeType int

const (
	// complete names and IDs after two chars
	completeDefault completeType = iota
	// only complete IDs
	completeIDs
	// only complete Names
	completeNames
)

type keyValueCompletion map[string]func(s string) ([]string, cobra.ShellCompDirective)

func getContainers(toComplete string, cType completeType, statuses ...string) ([]string, cobra.ShellCompDirective) {
	suggestions := []string{}
	listOpts := entities.ContainerListOptions{
		Filters: make(map[string][]string),
	}
	listOpts.All = true
	listOpts.Pod = true
	if len(statuses) > 0 {
		listOpts.Filters["status"] = statuses
	}

	containers, err := registry.ContainerEngine().ContainerList(registry.GetContext(), listOpts)
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	for _, c := range containers {
		// include ids in suggestions if cType == completeIDs or
		// more then 2 chars are typed and cType == completeDefault
		if ((len(toComplete) > 1 && cType == completeDefault) ||
			cType == completeIDs) && strings.HasPrefix(c.ID, toComplete) {
			suggestions = append(suggestions, c.ID[0:12]+"\t"+c.PodName)
		}
		// include name in suggestions
		if cType != completeIDs && strings.HasPrefix(c.Names[0], toComplete) {
			suggestions = append(suggestions, c.Names[0]+"\t"+c.PodName)
		}
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

func getPods(toComplete string, cType completeType, statuses ...string) ([]string, cobra.ShellCompDirective) {
	suggestions := []string{}
	listOpts := entities.PodPSOptions{
		Filters: make(map[string][]string),
	}
	if len(statuses) > 0 {
		listOpts.Filters["status"] = statuses
	}

	pods, err := registry.ContainerEngine().PodPs(registry.GetContext(), listOpts)
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	for _, pod := range pods {
		// include ids in suggestions if cType == completeIDs or
		// more then 2 chars are typed and cType == completeDefault
		if ((len(toComplete) > 1 && cType == completeDefault) ||
			cType == completeIDs) && strings.HasPrefix(pod.Id, toComplete) {
			suggestions = append(suggestions, pod.Id[0:12])
		}
		// include name in suggestions
		if cType != completeIDs && strings.HasPrefix(pod.Name, toComplete) {
			suggestions = append(suggestions, pod.Name)
		}
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

func getVolumes(toComplete string) ([]string, cobra.ShellCompDirective) {
	suggestions := []string{}
	lsOpts := entities.VolumeListOptions{}

	volumes, err := registry.ContainerEngine().VolumeList(registry.GetContext(), lsOpts)
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	for _, v := range volumes {
		if strings.HasPrefix(v.Name, toComplete) {
			suggestions = append(suggestions, v.Name)
		}
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

func getImages(toComplete string) ([]string, cobra.ShellCompDirective) {
	suggestions := []string{}
	listOptions := entities.ImageListOptions{}

	images, err := registry.ImageEngine().List(registry.GetContext(), listOptions)
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	for _, image := range images {
		// include ids in suggestions if more then 2 chars are typed
		if len(toComplete) > 1 && strings.HasPrefix(image.ID, toComplete) {
			suggestions = append(suggestions, image.ID[0:12])
		}
		for _, repo := range image.RepoTags {
			if toComplete == "" {
				// suggest only full repo path if no input is given
				if strings.HasPrefix(repo, toComplete) {
					suggestions = append(suggestions, repo)
				}
			} else {
				// suggested "registry.fedoraproject.org/f29/httpd:latest" as
				// - "registry.fedoraproject.org/f29/httpd:latest"
				// - "registry.fedoraproject.org/f29/httpd"
				// - "f29/httpd:latest"
				// - "f29/httpd"
				// - "httpd:latest"
				// - "httpd"
				paths := strings.Split(repo, "/")
				for i := range paths {
					suggestionWithTag := strings.Join(paths[i:], "/")
					if strings.HasPrefix(suggestionWithTag, toComplete) {
						suggestions = append(suggestions, suggestionWithTag)
					}
					suggestionWithoutTag := strings.SplitN(strings.SplitN(suggestionWithTag, ":", 2)[0], "@", 2)[0]
					if strings.HasPrefix(suggestionWithoutTag, toComplete) {
						suggestions = append(suggestions, suggestionWithoutTag)
					}
				}
			}
		}
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

func getRegistries() ([]string, cobra.ShellCompDirective) {
	regs, err := registries.GetRegistries()
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}
	return regs, cobra.ShellCompDirectiveNoFileComp
}

func getNetworks(toComplete string) ([]string, cobra.ShellCompDirective) {
	suggestions := []string{}
	networkListOptions := entities.NetworkListOptions{}

	networks, err := registry.ContainerEngine().NetworkList(registry.Context(), networkListOptions)
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	for _, n := range networks {
		if strings.HasPrefix(n.Name, toComplete) {
			suggestions = append(suggestions, n.Name)
		}
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// validCurrentCmdLine validates the current cmd line
// It utilizes the Args function from the cmd struct
// In most cases the Args function validates the args length but it
// is also used to verify that --latest is not given with an argument.
// This function helps to makes sure we only complete valid arguments.
func validCurrentCmdLine(cmd *cobra.Command, args []string, toComplete string) bool {
	if cmd.Args == nil {
		// Without an Args function we cannot check so assume it's correct
		return true
	}
	// We have to append toComplete to the args otherwise the
	// argument count would not match the expected behavior
	if err := cmd.Args(cmd, append(args, toComplete)); err != nil {
		// Special case if we use ExactArgs(2) or MinimumNArgs(2),
		// They will error if we try to complete the first arg.
		// Lets try to parse the common error and compare if we have less args than
		// required. In this case we are fine and should provide completion.

		// Clean the err msg so we can parse it with fmt.Sscanf
		// Trim MinimumNArgs prefix
		cleanErr := strings.TrimPrefix(err.Error(), "requires at least ")
		// Trim MinimumNArgs "only" part
		cleanErr = strings.ReplaceAll(cleanErr, "only received", "received")
		// Trim ExactArgs prefix
		cleanErr = strings.TrimPrefix(cleanErr, "accepts ")
		var need, got int
		cobra.CompDebugln(cleanErr, true)
		_, err = fmt.Sscanf(cleanErr, "%d arg(s), received %d", &need, &got)
		if err == nil {
			if need >= got {
				// We still need more arguments so provide more completions
				return true
			}
		}
		cobra.CompDebugln(err.Error(), true)
		return false
	}
	return true
}

func prefixSlice(pre string, slice []string) []string {
	for i := range slice {
		slice[i] = pre + slice[i]
	}
	return slice
}

func completeKeyValues(toComplete string, k keyValueCompletion) ([]string, cobra.ShellCompDirective) {
	suggestions := make([]string, 0, len(k))
	directive := cobra.ShellCompDirectiveNoFileComp
	for key, getComps := range k {
		if strings.HasPrefix(toComplete, key) {
			if getComps != nil {
				suggestions, dir := getComps(toComplete[len(key):])
				return prefixSlice(key, suggestions), dir
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if strings.HasPrefix(key, toComplete) {
			suggestions = append(suggestions, key)
			latKey := key[len(key)-1:]
			if latKey == "=" || latKey == ":" {
				// make sure we don't add a space after ':' or '='
				directive = cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
			}
		}
	}
	return suggestions, directive
}

/* Autocomplete Functions for cobra ValidArgsFunction */

// AutocompleteContainers - Autocomplete all container names.
func AutocompleteContainers(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return getContainers(toComplete, completeDefault)
}

// AutocompleteContainersCreated - Autocomplete only created container names.
func AutocompleteContainersCreated(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return getContainers(toComplete, completeDefault, "created")
}

// AutocompleteContainersExited - Autocomplete only exited container names.
func AutocompleteContainersExited(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return getContainers(toComplete, completeDefault, "exited")
}

// AutocompleteContainersPaused - Autocomplete only paused container names.
func AutocompleteContainersPaused(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return getContainers(toComplete, completeDefault, "paused")
}

// AutocompleteContainersRunning - Autocomplete only running container names.
func AutocompleteContainersRunning(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return getContainers(toComplete, completeDefault, "running")
}

// AutocompleteContainersStartable - Autocomplete only created and exited container names.
func AutocompleteContainersStartable(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return getContainers(toComplete, completeDefault, "created", "exited")
}

// AutocompletePods - Autocomplete all pod names.
func AutocompletePods(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return getPods(toComplete, completeDefault)
}

// AutocompletePodsRunning - Autocomplete only running pod names.
// It considers degraded as running.
func AutocompletePodsRunning(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return getPods(toComplete, completeDefault, "running", "degraded")
}

// AutocompleteContainersAndPods - Autocomplete container names and pod names.
func AutocompleteContainersAndPods(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	containers, _ := getContainers(toComplete, completeDefault)
	pods, _ := getPods(toComplete, completeDefault)
	return append(containers, pods...), cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteContainersAndImages - Autocomplete container names and pod names.
func AutocompleteContainersAndImages(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	containers, _ := getContainers(toComplete, completeDefault)
	images, _ := getImages(toComplete)
	return append(containers, images...), cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteVolumes - Autocomplete volumes.
func AutocompleteVolumes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return getVolumes(toComplete)
}

// AutocompleteImages - Autocomplete images.
func AutocompleteImages(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return getImages(toComplete)
}

// AutocompleteCreateRun - Autocomplete only the fist argument as image and then do file completion.
func AutocompleteCreateRun(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if len(args) < 1 {
		return getImages(toComplete)
	}
	// TODO: add path completion for files in the image
	return nil, cobra.ShellCompDirectiveDefault
}

// AutocompleteRegistries - Autocomplete registries.
func AutocompleteRegistries(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return getRegistries()
}

// AutocompleteNetworks - Autocomplete networks.
func AutocompleteNetworks(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return getNetworks(toComplete)
}

// AutocompleteCpCommand - Autocomplete podman cp command args.
func AutocompleteCpCommand(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if len(args) < 2 {
		containers, _ := getContainers(toComplete, completeDefault)
		for _, container := range containers {
			// TODO: Add path completion for inside the container if possible
			if strings.HasPrefix(container, toComplete) {
				return containers, cobra.ShellCompDirectiveNoSpace
			}
		}
		// else complete paths
		return nil, cobra.ShellCompDirectiveDefault
	}
	// don't complete more than 2 args
	return nil, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteSystemConnections - Autocomplete system connections.
func AutocompleteSystemConnections(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !validCurrentCmdLine(cmd, args, toComplete) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	suggestions := []string{}
	cfg, err := config.ReadCustomConfig()
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	for k, v := range cfg.Engine.ServiceDestinations {
		// the URI will be show as description in shells like zsh
		suggestions = append(suggestions, k+"\t"+v.URI)
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

/* -------------- Flags ----------------- */

// AutocompleteDetachKeys - Autocomplete detach-keys options.
// -> "ctrl-"
func AutocompleteDetachKeys(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if strings.HasSuffix(toComplete, ",") {
		return []string{toComplete + "ctrl-"}, cobra.ShellCompDirectiveNoSpace
	}
	return []string{"ctrl-"}, cobra.ShellCompDirectiveNoSpace
}

// AutocompleteChangeInstructions - Autocomplete change instructions options for commit and import.
// -> "CMD", "ENTRYPOINT", "ENV", "EXPOSE", "LABEL", "ONBUILD", "STOPSIGNAL", "USER", "VOLUME", "WORKDIR"
func AutocompleteChangeInstructions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return ChangeCmds, cobra.ShellCompDirectiveNoSpace
}

// AutocompleteImageFormat - Autocomplete image format options.
// -> "oci", "docker"
func AutocompleteImageFormat(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ImageFormat := []string{"oci", "docker"}
	return ImageFormat, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteCreateAttach - Autocomplete create --attach options.
// -> "stdin", "stdout", "stderr"
func AutocompleteCreateAttach(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"stdin", "stdout", "stderr"}, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteNamespace - Autocomplete namespace options.
// -> host,container:[name],ns:[path],private
func AutocompleteNamespace(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	kv := keyValueCompletion{
		"container:": func(s string) ([]string, cobra.ShellCompDirective) { return getContainers(s, completeDefault) },
		"ns:":        func(s string) ([]string, cobra.ShellCompDirective) { return nil, cobra.ShellCompDirectiveDefault },
		"host":       nil,
		"private":    nil,
	}
	return completeKeyValues(toComplete, kv)
}

// AutocompleteUserNamespace - Autocomplete namespace options.
// -> same as AutocompleteNamespace with "auto", "keep-id" added
func AutocompleteUserNamespace(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	results, directive := AutocompleteNamespace(cmd, args, toComplete)
	results = append(results, "auto", "keep-id")
	return results, directive
}

// AutocompleteCgroupMode - Autocomplete cgroup mode options.
// -> "enabled", "disabled", "no-conmon", "split"
func AutocompleteCgroupMode(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cgroupModes := []string{"enabled", "disabled", "no-conmon", "split"}
	return cgroupModes, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteImageVolume - Autocomplete image volume options.
// -> "bind", "tmpfs", "ignore"
func AutocompleteImageVolume(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	imageVolumes := []string{"bind", "tmpfs", "ignore"}
	return imageVolumes, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteLogDriver - Autocomplete log-driver options.
// -> "journald", "none", "k8s-file"
func AutocompleteLogDriver(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// don't show json-file
	logDrivers := []string{define.JournaldLogging, define.NoLogging, define.KubernetesLogging}
	return logDrivers, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteLogOpt - Autocomplete log-opt options.
// -> "path=", "tag="
func AutocompleteLogOpt(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// FIXME: are these the only one? the man page states these but in the current shell completion they are more options
	logOptions := []string{"path=", "tag="}
	if strings.HasPrefix(toComplete, "path=") {
		return nil, cobra.ShellCompDirectiveDefault
	}
	return logOptions, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}

// AutocompletePullOption - Autocomplete pull options for create and run command.
// -> "always", "missing", "never"
func AutocompletePullOption(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pullOptions := []string{"always", "missing", "never"}
	return pullOptions, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteRestartOption - Autocomplete restart options for create and run command.
// -> "always", "no", "on-failure", "unless-stopped"
func AutocompleteRestartOption(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	restartOptions := []string{libpod.RestartPolicyAlways, libpod.RestartPolicyNo,
		libpod.RestartPolicyOnFailure, libpod.RestartPolicyUnlessStopped}
	return restartOptions, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteSecurityOption - Autocomplete security options options.
func AutocompleteSecurityOption(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	kv := keyValueCompletion{
		"apparmor=":         nil,
		"no-new-privileges": nil,
		"seccomp=":          func(s string) ([]string, cobra.ShellCompDirective) { return nil, cobra.ShellCompDirectiveDefault },
		"label=": func(s string) ([]string, cobra.ShellCompDirective) {
			if strings.HasPrefix(s, "d") {
				return []string{"disable"}, cobra.ShellCompDirectiveNoFileComp
			}
			return []string{"user:", "role:", "type:", "level:", "filetype:", "disable"}, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		},
	}
	return completeKeyValues(toComplete, kv)
}

// AutocompleteStopSignal - Autocomplete stop signal options.
// -> "SIGHUP", "SIGINT", "SIGKILL", "SIGTERM"
func AutocompleteStopSignal(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// FIXME: add more/different signals?
	stopSignals := []string{"SIGHUP", "SIGINT", "SIGKILL", "SIGTERM"}
	return stopSignals, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteSystemdFlag - Autocomplete systemd flag options.
// -> "true", "false", "always"
func AutocompleteSystemdFlag(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	systemd := []string{"true", "false", "always"}
	return systemd, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteUserFlag - Autocomplete user flag based on the names and groups (includes ids after first char) in /etc/passwd and /etc/group files.
// -> user:group
func AutocompleteUserFlag(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if strings.Contains(toComplete, ":") {
		// It would be nice to read the file in the image
		// but at this point we don't know the image.
		file, err := os.Open("/etc/group")
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		defer file.Close()

		var groups []string
		scanner := bufio.NewScanner(file)
		user := strings.SplitN(toComplete, ":", 2)[0]
		for scanner.Scan() {
			entries := strings.SplitN(scanner.Text(), ":", 4)
			groups = append(groups, user+":"+entries[0])
			// complete ids after at least one char is given
			if len(user)+1 < len(toComplete) {
				groups = append(groups, user+":"+entries[2])
			}
		}
		if err = scanner.Err(); err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return groups, cobra.ShellCompDirectiveNoFileComp
	}

	// It would be nice to read the file in the image
	// but at this point we don't know the image.
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	defer file.Close()

	var users []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		entries := strings.SplitN(scanner.Text(), ":", 7)
		users = append(users, entries[0]+":")
		// complete ids after at least one char is given
		if len(toComplete) > 0 {
			users = append(users, entries[2]+":")
		}
	}
	if err = scanner.Err(); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return users, cobra.ShellCompDirectiveNoSpace
}

// AutocompleteMountFlag - Autocomplete mount flag options.
// -> "type=bind,", "type=volume,", "type=tmpfs,"
func AutocompleteMountFlag(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	types := []string{"type=bind,", "type=volume,", "type=tmpfs,"}
	// TODO: Add support for all different options
	return types, cobra.ShellCompDirectiveNoSpace
}

// AutocompleteVolumeFlag - Autocomplete volume flag options.
// -> volumes and paths
func AutocompleteVolumeFlag(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	volumes, _ := getVolumes(toComplete)
	directive := cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveDefault
	if strings.Contains(toComplete, ":") {
		// add space after second path
		directive = cobra.ShellCompDirectiveDefault
	}
	return volumes, directive
}

// AutocompleteJSONFormat - Autocomplete format flag option.
// -> "json"
func AutocompleteJSONFormat(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"json"}, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteEventFilter - Autocomplete event filter flag options.
// -> "container=", "event=", "image=", "pod=", "volume=", "type="
func AutocompleteEventFilter(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	filters := []string{"container=", "event=", "image=", "pod=", "volume=", "type="}
	return filters, cobra.ShellCompDirectiveNoSpace
}

// AutocompleteSystemdRestartOptions - Autocomplete systemd restart options.
// -> "no", "on-success", "on-failure", "on-abnormal", "on-watchdog", "on-abort", "always"
func AutocompleteSystemdRestartOptions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return systemdGen.RestartPolicies, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteTrustType - Autocomplete trust type options.
// -> "signedBy", "accept", "reject"
func AutocompleteTrustType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	types := []string{"signedBy", "accept", "reject"}
	return types, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteImageSort - Autocomplete images sort options.
// -> "created", "id", "repository", "size", "tag"
func AutocompleteImageSort(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	sortBy := []string{"created", "id", "repository", "size", "tag"}
	return sortBy, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteInspectType - Autocomplete inspect type options.
// -> "container", "image", "all"
func AutocompleteInspectType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	types := []string{"container", "image", "all"}
	return types, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteManifestFormat - Autocomplete manifest format options.
// -> "oci", "v2s2"
func AutocompleteManifestFormat(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	types := []string{"oci", "v2s2"}
	return types, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteNetworkDriver - Autocomplete network driver option.
// -> "bridge"
func AutocompleteNetworkDriver(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	drivers := []string{"bridge"}
	return drivers, cobra.ShellCompDirectiveNoFileComp
}

// AutocompletePodShareNamespace - Autocomplete pod create --share flag option.
// -> "ipc", "net", "pid", "user", "uts", "cgroup", "none"
func AutocompletePodShareNamespace(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	namespaces := []string{"ipc", "net", "pid", "user", "uts", "cgroup", "none"}
	return namespaces, cobra.ShellCompDirectiveNoFileComp
}

// AutocompletePodPsSort - Autocomplete images sort options.
// -> "created", "id", "name", "status", "number"
func AutocompletePodPsSort(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	sortBy := []string{"created", "id", "name", "status", "number"}
	return sortBy, cobra.ShellCompDirectiveNoFileComp
}

// AutocompletePsSort - Autocomplete images sort options.
// -> "command", "created", "id", "image", "names", "runningfor", "size", "status"
func AutocompletePsSort(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	sortBy := []string{"command", "created", "id", "image", "names", "runningfor", "size", "status"}
	return sortBy, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteImageSaveFormat - Autocomplete image save format options.
// -> "oci-archive", "oci-dir", "docker-dir"
func AutocompleteImageSaveFormat(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	formats := []string{"oci-archive", "oci-dir", "docker-dir"}
	return formats, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteWaitCondition - Autocomplete wait condition options.
// -> "unknown", "configured", "created", "running", "stopped", "paused", "exited", "removing"
func AutocompleteWaitCondition(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	states := []string{"unknown", "configured", "created", "running", "stopped", "paused", "exited", "removing"}
	return states, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteCgroupManager - Autocomplete cgroup manager options.
// -> "cgroupfs", "systemd"
func AutocompleteCgroupManager(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	types := []string{"cgroupfs", "systemd"}
	return types, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteEventBackend - Autocomplete event backend options.
// -> "file", "journald", "none"
func AutocompleteEventBackend(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	types := []string{"file", "journald", "none"}
	return types, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteLogLevel - Autocomplete log level options.
// -> "debug", "info", "warn", "error", "fatal", "panic"
func AutocompleteLogLevel(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return LogLevels, cobra.ShellCompDirectiveNoFileComp
}

// AutocompleteSDNotify - Autocomplete sdnotify options.
// -> "container", "conmon", "ignore"
func AutocompleteSDNotify(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	types := []string{"container", "conmon", "ignore"}
	return types, cobra.ShellCompDirectiveNoFileComp
}

var containerStatuses = []string{"created", "running", "paused", "stopped", "exited", "unknown"}

// AutocompletePsFilters - Autocomplete ps filter options.
func AutocompletePsFilters(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	kv := keyValueCompletion{
		"id=":   func(s string) ([]string, cobra.ShellCompDirective) { return getContainers(s, completeIDs) },
		"name=": func(s string) ([]string, cobra.ShellCompDirective) { return getContainers(s, completeNames) },
		"status=": func(_ string) ([]string, cobra.ShellCompDirective) {
			return containerStatuses, cobra.ShellCompDirectiveNoFileComp
		},
		"ancestor": func(s string) ([]string, cobra.ShellCompDirective) { return getImages(s) },
		"before=":  func(s string) ([]string, cobra.ShellCompDirective) { return getContainers(s, completeDefault) },
		"since=":   func(s string) ([]string, cobra.ShellCompDirective) { return getContainers(s, completeDefault) },
		"volume=":  func(s string) ([]string, cobra.ShellCompDirective) { return getVolumes(s) },
		"health=": func(_ string) ([]string, cobra.ShellCompDirective) {
			return []string{define.HealthCheckHealthy,
				define.HealthCheckUnhealthy}, cobra.ShellCompDirectiveNoFileComp
		},
		"label=":  nil,
		"exited=": nil,
		"until=":  nil,
	}
	return completeKeyValues(toComplete, kv)
}

// AutocompletePodPsFilters - Autocomplete pod ps filter options.
func AutocompletePodPsFilters(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	kv := keyValueCompletion{
		"id=":   func(s string) ([]string, cobra.ShellCompDirective) { return getPods(s, completeIDs) },
		"name=": func(s string) ([]string, cobra.ShellCompDirective) { return getPods(s, completeNames) },
		"status=": func(_ string) ([]string, cobra.ShellCompDirective) {
			return []string{"stopped", "running",
				"paused", "exited", "dead", "created", "degraded"}, cobra.ShellCompDirectiveNoFileComp
		},
		"ctr-ids=":    func(s string) ([]string, cobra.ShellCompDirective) { return getContainers(s, completeIDs) },
		"ctr-names=":  func(s string) ([]string, cobra.ShellCompDirective) { return getContainers(s, completeNames) },
		"ctr-number=": nil,
		"ctr-status=": func(_ string) ([]string, cobra.ShellCompDirective) {
			return containerStatuses, cobra.ShellCompDirectiveNoFileComp
		},
		"label=": nil,
	}
	return completeKeyValues(toComplete, kv)
}
