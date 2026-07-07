## Sharing Outside Your Tailnet

AgentHub gives you two ways to share a session with someone who is not on your Tailscale network. **Option 1 — Tailscale Funnel** publishes the session on the public internet, gated by a join code that expires. **Option 2 — Device Share + ACL** keeps traffic inside Tailscale by sharing your device with a specific person and granting them access only to the AgentHub port.

### Option 1 — Tailscale Funnel (public internet)

Funnel lets anyone with the link reach your session over the internet — no Tailscale account required for the guest.

1. Open the **Share Session** modal from the status bar at the bottom of the window.
2. Under **Internet (public)**, toggle **Enable internet share** on.
3. Read the risk dialog and choose an expiry — 30 minutes, 1 hour (default), 4 hours, 8 hours, or "Until I disable" (no auto-expiry). The Funnel link becomes public the moment you confirm.
4. Copy the public URL from the modal and send it to your guest together with the join code displayed below it.

The **join code is the only access gate**. Anyone who has both the URL and the join code can view the session. Codes expire when the Funnel expires or when you disable internet sharing — whichever comes first.

To stop sharing, toggle **Enable internet share** off in the modal. AgentHub immediately tears down the Funnel and the public URL stops working.

For more on how Tailscale Funnel works, see the [Tailscale Funnel documentation](https://tailscale.com/kb/1223/tailscale-funnel/).

### Public write access (FULL ACCESS)

By default, a Funnel link only grants **read-only** viewing. Public write access is a separate, opt-in escalation available once internet sharing is on.

**What it grants — stated plainly, not softened: command execution.** Enabling public write access lets anyone who has the write link and the single-use write code type into your terminal and run commands, exactly as if they were sitting at your keyboard. This is a real remote-code-execution risk. Only enable it when you specifically intend to hand over terminal control to someone.

**How to enable it.** In the Share modal, scroll to the **Danger** section. Read the warning copy, choose an expiry (15 minutes, 30 minutes, or 1 hour — there is no "until I disable" option for public write), and press and hold the confirm button for 3 seconds. Releasing early cancels the action and nothing is granted — the hold is a deliberate friction gate, not a decoration.

**The write code is single-use, one writer, short-lived.** Once someone redeems the write code, it cannot be redeemed again — there is exactly one writer per grant. The default lifetime is 15 minutes; the maximum is 1 hour. When the timer runs out, write access ends automatically.

**Disabling is one click and immediate.** Click "Disable public write" in the Danger section at any time. The writer is cut off instantly — their next keystroke will not reach the terminal.

**The read share is unaffected.** Disabling public write access, or letting it expire, does not touch the separate read-only public link. A read spectator using the reusable read code keeps viewing without interruption.

### Option 2 — Device Share + ACL (contained)

Device sharing keeps traffic inside the Tailscale network. Your guest must have a Tailscale account. This option gives you fine-grained control over who can connect.

**Step 1 — Share your device.**
Go to the [Tailscale admin console](https://login.tailscale.com/admin/machines), find your machine, and use **Share → Share with...** to invite the person. They will receive an email to accept the share. See the [Tailscale sharing guide](https://tailscale.com/kb/1084/sharing/) for the full walkthrough.

**Step 2 — Tag your device.**
Assign the tag `tag:agenthub` to your machine in the Tailscale admin console. Tags let you write ACL rules that target the device by role rather than by IP. See [ACL tags](https://tailscale.com/kb/1068/acl-tags/) for setup instructions.

**Step 3 — Grant access to the AgentHub port.**
Add the following rule to your Tailscale ACL policy (under **Access controls** in the admin console). The rule allows shared users (`autogroup:shared`) to reach only the AgentHub web port (TCP 7443) on your tagged machine:

```json
{
  "action": "accept",
  "src":    ["autogroup:shared"],
  "dst":    ["tag:agenthub:tcp:7443"]
}
```

> **Important — wildcard-default gotcha:** Tailscale ACLs default to `*→*` (allow all). Adding a `tag:agenthub` restriction will narrow this — verify your full ACL after editing.

Without an explicit deny, a shared user who is also in an `allow *→*` rule can still reach other ports on your machine. Always review your full ACL in the admin console after making changes. See [ACL syntax reference](https://tailscale.com/kb/1337/acl-syntax/) for the full rule schema.

**Sharing the session URL.**
Once the device is shared and the ACL rule is in place, the guest can open AgentHub's local web address (`https://<your-machine>.ts.net:7443`) in a browser on their device. No join code is required — access is controlled by the Tailscale ACL.

If you want to share with someone who does not have a Tailscale account, use **Option 1 — Tailscale Funnel** instead.
