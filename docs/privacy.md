# Privacy Policy

**Last updated:** 2026-10-20

This privacy policy applies to the **Get offa Windows** / **Save Channeler** desktop application and related Google OAuth clients used with it (the “App”), maintained at [github.com/migzeetigglez/ashen-sync](https://github.com/migzeetigglez/ashen-sync).

## Summary

The App runs on **your** device. It helps you copy files you choose (such as game saves) to cloud storage **you** control, using [rclone](https://rclone.org/). We do not operate a sync server that receives your files or Google account data.

## Google user data

When you authorize Google Drive access (via the App or rclone configured with this project’s OAuth client), the App / rclone may access:

- **Files and folders** in Google Drive that you grant access to (names, sizes, modification times, and file contents needed to upload, download, list, or sync).
- **OAuth tokens** issued by Google so the App can call the Drive API on your behalf.
- **Basic account identifiers** Google returns during sign-in (for example, email), used only to label the connected account in local configuration.

### How that data is used

- Solely to provide local file sync / transfer features you initiate (list, upload, download, sync).
- Use of Google user data is limited to these practices and complies with Google’s [Limited Use requirements](https://developers.google.com/terms/api-services-user-data-policy) for the scopes you approve.

### How that data is stored

- OAuth tokens and remote configuration are stored in **local files on your machine** (for example, rclone’s config under your user profile).
- The App does not upload your Google tokens or Drive file contents to servers run by this project.

### Sharing

- We do **not** sell Google user data.
- We do **not** share Google user data with third parties for advertising, analytics, or unrelated services.
- Data stays between your device, Google’s APIs, and (if you choose another provider) that provider—according to their policies.

## Other cloud providers

If you use Proton Drive or another rclone backend, that provider’s privacy policy applies to data you store there. This document focuses on Google OAuth / Drive usage required for Google Cloud consent branding.

## No telemetry

The App does not include telemetry that phones home usage analytics to this project.

## Children’s privacy

The App is not directed at children under 13.

## Changes

We may update this policy by changing this page. The “Last updated” date will change when we do.

## Contact

Questions about this policy: open an issue on [github.com/migzeetigglez/ashen-sync](https://github.com/migzeetigglez/ashen-sync/issues) or email the support address shown on the Google OAuth consent screen for this app (`miketooantuh@gmail.com`).
