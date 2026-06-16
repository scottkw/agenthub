---
phase: quick
plan: 260406-op4
status: complete
one_liner: "Tray icon replaced with monochrome silhouette of app icon A letterform extracted from title logo"
---

## Summary

Replaced the programmatically generated geometric tray icon with a monochrome silhouette extracted from the actual app logo (`frontend/src/assets/agenthub-title-logo.png`). The process:

1. Crop the "A" from the title logo (185x208 left portion)
2. Clean background to pure white
3. Create grayscale mask via threshold (dark A pixels → opaque, white → transparent)
4. Composite black fill through the mask → black A on transparent background
5. Resize to 18x18 for macOS tray template icon

The tray icon now matches the app icon's distinctive "A" with dot letterform. macOS `setTemplate:YES` handles light/dark adaptation. Error tray icon (circle-exclamation) unchanged as it serves a different purpose.
