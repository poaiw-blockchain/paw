# Browser Extension Submission Guide

This guide captures the exact build artifacts, metadata, and review steps needed to publish the PAW Browser Wallet & Miner extension across Chrome, Firefox, and Edge stores.

## 1. Preparation Checklist

1. **Install & Build**
   ```bash
   npm install
   npm run lint
   npm run package
   ```
   - `npm run package` produces `extension.zip` in the project root.
2. **Assets**
   - `dist/manifest.json` → confirm `version`, `name`, `description`, and permissions match store requirements.
   - Icons: ensure the `icons` array in the manifest references 128x128 (Chrome/Edge) and 48x48/96x96 (Firefox) assets.
3. **Documentation**
   - Update the store listing descriptions (features, privacy practices, contact email).
   - Capture screenshots/GIFs from the latest build (popup, config screen, mining tab).
4. **Security**
   - Confirm `npm run lint` and `npm audit` are clean or document any remaining advisories.
   - Validate no external network calls occur outside the configured API host.

## 2. Chrome Web Store

1. Navigate to <https://chrome.google.com/webstore/devconsole>.
2. Create a new item (or update existing) and upload `extension.zip`.
3. Fill in:
   - **Category**: Productivity → Finance (suggested).
   - **Short description**: "PAW browser wallet with mining, swaps, and monitoring."
   - **Privacy**: declare that no personal data is collected/stored.
   - **Permissions justification**: match the manifest (`storage`, `activeTab`, network hosts).
4. Provide screenshots (1280×800 or above) plus a 128x128 icon.
5. Submit for review; track the status from the Dev Console.

## 3. Firefox Add-ons (AMO)

1. Go to <https://addons.mozilla.org/developers/>.
2. Select **Submit a New Add-on** → **Upload Existing**.
3. Upload `extension.zip`.
4. Complete metadata:
   - Compatible application: Firefox / Firefox ESR.
   - Categories: "Privacy & Security" → "Crypto".
   - Provide translated descriptions if desired.
5. AMO signs the extension; download the `.xpi` after review for distribution.

## 4. Microsoft Edge Add-ons

1. Visit <https://partner.microsoft.com/en-us/dashboard/microsoftedge/>.
2. Create a new submission, upload `extension.zip`.
3. Reuse the Chrome metadata; Edge accepts Manifest V3 packages.
4. Add a privacy policy link (recommended).
5. Submit; the Edge team reuses Chrome reviews if the same package hash is detected.

## 5. Post-Submission

- Track review statuses in each portal; note any policy issues inside `ROADMAP_PRODUCTION.md`.
- Once approved, tag the commit and record the published store URLs for reference in documentation.
