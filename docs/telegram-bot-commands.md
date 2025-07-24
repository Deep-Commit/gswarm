# G-Swarm Telegram Bot Commands

The G-Swarm Telegram bot provides several commands to help you monitor and manage your Gensyn AI node activity.

## Available Commands

### `/stats`
**Purpose:** Check your current stats and rank data on demand

**Usage:** Simply type `/stats` in the Telegram chat

**What it shows:**
- Verification status (verified/unverified)
- Total nodes and ranked nodes (for verified users)
- EOA address being monitored
- Number of peer IDs being tracked
- Total votes and rewards across all peers
- Per-peer breakdown with individual stats
- Rank information for each peer (if verified)
- Real-time timestamp of when data was fetched

**Example output:**
```
📊 G-Swarm Stats Report

✅ Verified User
📊 Total Nodes: 3
🏆 Ranked Nodes: 2

👤 EOA Address: 0x1234...5678
🔍 Peer IDs Monitored: 3

📈 Total Votes: 1500
💰 Total Rewards: 250

📋 Per-Peer Breakdown:
🔹 Peer 1: 12D...3K4
   📈 Votes: 500
   💰 Rewards: 100
   🏆 Rank: #15

🔹 Peer 2: 45E...7L8
   📈 Votes: 600
   💰 Rewards: 120
   🏆 Rank: #12

🔹 Peer 3: 78F...9M0
   📈 Votes: 400
   💰 Rewards: 30

⏰ Last Updated: 2025-01-27 14:30:25

💡 Stats are fetched in real-time. Use this command anytime to check your current status!
```

### `/verify <code>`
**Purpose:** Link your Discord and Telegram accounts

**Usage:** `/verify <verification_code>`

**Process:**
1. Get a verification code from Discord using `/link-telegram`
2. Use the code with this command in Telegram
3. Your accounts will be linked automatically

### `/help`
**Purpose:** Show available commands and basic information

**Usage:** Simply type `/help`

### `/start`
**Purpose:** Start the bot (no response sent)

**Usage:** Automatically triggered when starting a conversation

## Benefits of the /stats Command

1. **On-demand access:** Check your stats anytime without waiting for hourly updates
2. **Real-time data:** Always get the most current information
3. **Comprehensive view:** See both blockchain data and rank information in one place
4. **Troubleshooting:** Verify your setup is working correctly
5. **User engagement:** Provides immediate feedback and keeps users informed

## Monitoring vs. Manual Checks

- **Automatic monitoring:** The bot checks your stats every hour and sends notifications only when there are changes
- **Manual stats check:** Use `/stats` anytime to get current information, regardless of whether there have been changes

This gives users the best of both worlds - automated monitoring for changes and manual access for immediate status checks. 