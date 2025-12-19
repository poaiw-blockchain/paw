# PAW Mobile Wallet – App Store Submission Checklist (iOS & Android)

Use this checklist for every release candidate before submitting to Apple App Store and Google Play.

## Common (Both Stores)
- [ ] Version bump & changelog
  - Update `package.json` version; ensure native version codes updated (`ios/PAWWallet/Info.plist` CFBundleShortVersionString/CFBundleVersion, `android/app/build.gradle` versionName/versionCode).
  - Update `CHANGELOG.md` with user-facing fixes/features.
- [ ] Build artifacts
  - iOS: Archive Release build (Generic iOS Device) → produce `.ipa` via Xcode Organizer.
  - Android: `cd android && ./gradlew clean bundleRelease assembleRelease` → outputs `.aab` + `app-release.apk`.
- [ ] Signing
  - iOS: Ensure correct distribution certificates/profiles, push notification entitlements, associated domains (if used).
  - Android: Sign with release keystore; verify `build.gradle` signing config uses secure env vars.
- [ ] Security/scans
  - Run `npm audit --production` (investigate blockers); run mobile unit tests `npm test`.
  - Verify no secrets/API keys in repo or bundles (`npm run lint` + spot-check generated outputs).
  - Confirm transport security: all API endpoints are https (or ATS exceptions documented).
- [ ] Assets & metadata
  - App name, icon, splash/screenshots up to date; status bar color matches brand.
  - Privacy policy URL and support URL set to project-approved endpoints.
  - In-app links to Terms/Privacy present in settings screen.
- [ ] QA sign-off
  - Execute `wallet/mobile/PLATFORM_TEST_PLAN.md` matrix; attach run log and crash-free evidence (Xcode/Play Console prelaunch reports if available).

## Apple App Store (App Store Connect)
- [ ] App Store record
  - Bundle ID matches provisioning profile; push capability enabled if notifications used.
  - Prepare “What’s New” text; category/keywords/localization updated.
  - Age rating questionnaire completed; encryption export compliance answered (uses standard HTTPS/crypto).
- [ ] Upload & validation
  - Deliver via Xcode Organizer or `xcrun altool`/Transporter; resolve any ITMS-90xxx errors.
  - Run App Store Connect validation: ATS compliance, UIRequiredDeviceCapabilities set (camera for QR, biometrics).
- [ ] TestFlight
  - Create external/internal group; upload release notes; include test account instructions.
  - Request review for External testing if needed; ensure compliance with Apple’s privacy nutrition label.
- [ ] App Review attachments
  - Provide demo credentials (non-privileged), screencast of key flows (create/import/send/staking/swap), and contact info.

## Google Play
- [ ] Play Console setup
  - Package name consistent; app signing key uploaded (Play App Signing recommended).
  - Content rating questionnaire completed; Data safety form updated for analytics/notifications.
  - Target SDK / min SDK align with store requirements; 64-bit native libs present if applicable.
- [ ] Upload
  - Upload `.aab` to new release; attach `.apk` for local sideload testing if desired.
  - Provide release notes (per track); select appropriate track (internal/closed/open/production).
- [ ] Pre-launch report
  - Enable Play’s automated pre-launch to capture crashes/screenshots on device matrix.
  - Review warnings (permissions, deep links, performance) and address blockers.

## Post-Submission
- [ ] Monitor crash/ANR dashboards (TestFlight metrics / Play Console vitals).
- [ ] Verify push notification delivery in production mode.
- [ ] Announce release with links and checksum/supply chain notes if distributing APK directly.
