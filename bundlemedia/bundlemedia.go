// Package bundlemedia documents HTTP metadata for Utilivo plugin bundle (.ubb) artifacts.
package bundlemedia

// PluginBundleContentType is the recommended Content-Type for .ubb payloads on upload
// and CDN storage. Do not use application/zip: clients often rename downloads to .zip.
const PluginBundleContentType = "application/octet-stream"
