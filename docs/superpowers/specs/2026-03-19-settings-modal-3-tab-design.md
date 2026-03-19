# Settings Modal — 3-Tab Reorganization

## Problem

The Web Serving tab bundles four distinct concerns (authentication, network config, server control, TLS certificates) into a single scrollable view. The layout feels jumbled and doesn't scale as settings grow.

## Solution

Split the 2-tab modal (CLI Paths, Web Serving) into 3 tabs by extracting security concerns into a dedicated tab.

## Tab Structure

| Tab | Default | Content |
|-----|---------|---------|
| CLI Paths | Yes | Path table with inline Save Paths button |
| Web Server | No | Description, network interface selector, port input, start/stop button, server URL display |
| Security | No | Dashboard password field + Set Password button, divider, CA certificate path + installation instructions |

## Component Changes

### SettingsPanel.tsx

1. **Tab state:** Change union type from `'cli-paths' | 'web-serving'` to `'cli-paths' | 'web-server' | 'security'`. Default remains `'cli-paths'`.

2. **Tab bar:** Add third button for "Security" tab with same pattern as existing two (role="tab", aria-selected, BEM active class).

3. **Body content:** Replace `activeTab === 'web-serving'` conditional with two new blocks:
   - `activeTab === 'web-server'`: description paragraph, network interface field-group, port field-group, start/stop button field-group with server URL display. The start button's existing `disabled` guard (`!isServerRunning && !isPasswordSet`) remains. Update the button's `title` attribute from "Set a password first" to "Set a password in the Security tab first" so users know where to go.
   - `activeTab === 'security'`: password field-group (label, input, Set Password button, error), horizontal rule divider (`<hr>` with inline style matching existing theme: `border:none; border-top:1px solid #292e42`), CA certificate field-group (label, description, cert path code block, installation details).

4. **Remove all h3 headings:** The tab buttons already label each section, so both `<h3>CLI Paths</h3>` and `<h3>Web Serving</h3>` are removed. The description paragraph ("Enable HTTPS access...") stays in Web Server as context.

5. **No new state variables:** All existing useState declarations and handler functions stay in the parent component. No new state needed.

6. **No footer changes:** Single Close button remains.

### style.css

No new CSS classes needed. The tab bar flexbox already accommodates N tabs. Several classes used by the component (e.g., `settings-panel__field-group`, `settings-panel__label`) are not defined in style.css — they rely on browser defaults. This is pre-existing and out of scope for this change.

## Test Changes

### Update existing tests
- Tests asserting 2 tab buttons → update to assert 3
- Change tab button text references from "Web Serving" to "Web Server"
- Update click-to-switch test to target "Web Server" instead of "Web Serving"

### Add new tests
- Test: Three tab buttons present with text "CLI Paths", "Web Server", "Security"
- Test: Clicking "Security" tab shows password field and CA certificate content
- Test: Security tab has `aria-selected="true"` on its tab button when active
- Test: Clicking "Security" tab hides CLI Paths and Web Server content
- Test: Web Server tab does NOT contain password input or CA certificate section
- Test: Security tab does NOT contain network interface selector or port input

## What stays the same

- All state variables in parent SettingsPanel component
- All handler functions (handleSaveCLIPaths, handleSetPassword, handleToggleServer, getCACertInstructions)
- Footer with single Close button
- Overlay click-to-close behavior
- JSX conditional rendering pattern (not CSS display toggling)
- BEM class naming convention
- Props interface (isOpen, onClose, clis)

## Files Modified

- `frontend/src/components/SettingsPanel.tsx`
- `frontend/src/components/__tests__/SettingsPanel.test.tsx`
