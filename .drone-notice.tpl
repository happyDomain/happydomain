happyDomain
===========

This product bundles third-party components under the following licenses.
The original license texts are reproduced in full below.

{{ range . }}
--------------------------------------------------------------------------------
Module:  {{ .Name }}
License: {{ .LicenseName }}
{{ with .LicenseURL }}Source:  {{ . }}
{{ end }}
{{ .LicenseText }}
{{ end }}
