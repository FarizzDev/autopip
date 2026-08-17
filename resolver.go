package main

func resolve(target string, kind Kind) bool {
	switch kind {
	case KindPip:
		step("Installing Python module via pip: %s", target)
		if err := run("pip", "install", target); err != nil {
			return false
		}
		success("Module '%s' installed.", target)
		return true

	case KindPkgDirect:
		pm, err := getSystemPackageManager()
		if err != nil {
			errlog("%v", err)
			return false
		}
		step("Installing package directly via %s: %s", pm.Name(), target)
		if err := pm.Install(target); err != nil {
			return false
		}
		success("Package '%s' installed. Retrying build...", target)
		return true

	default: // KindSystem
		return resolveSystemPackage(target)
	}
}
