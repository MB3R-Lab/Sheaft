{{- define "sheaft.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "sheaft.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "sheaft.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "sheaft.labels" -}}
app.kubernetes.io/name: {{ include "sheaft.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | quote }}
{{- end -}}

{{- define "sheaft.selectorLabels" -}}
app.kubernetes.io/name: {{ include "sheaft.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "sheaft.image" -}}
{{- if .Values.image.digest -}}
{{ printf "%s@%s" .Values.image.repository .Values.image.digest }}
{{- else -}}
{{ printf "%s:%s" .Values.image.repository .Values.image.tag }}
{{- end -}}
{{- end -}}

{{- define "sheaft.generatedConfigMapName" -}}
{{ printf "%s-config" (include "sheaft.fullname" .) }}
{{- end -}}

{{- define "sheaft.configMapName" -}}
{{- if .Values.config.existingConfigMap -}}
{{- .Values.config.existingConfigMap -}}
{{- else -}}
{{- include "sheaft.generatedConfigMapName" . -}}
{{- end -}}
{{- end -}}

{{- define "sheaft.hasInlineConfig" -}}
{{- if or .Values.config.analysis .Values.config.policy .Values.config.serve .Values.config.journeys -}}true{{- end -}}
{{- end -}}

{{- define "sheaft.volumeSource" -}}
{{- if eq .volume.type "pvc" -}}
persistentVolumeClaim:
  claimName: {{ .volume.existingClaim | quote }}
{{- else if eq .volume.type "hostPath" -}}
hostPath:
  path: {{ .volume.hostPath | quote }}
{{- else -}}
emptyDir: {}
{{- end -}}
{{- end -}}
