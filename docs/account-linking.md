# Secure Cross-Platform Account Linking

This document describes the secure account linking system that allows users to link their Discord and Telegram accounts for the G-Swarm platform.

## Overview

The account linking system provides a secure way for users to connect their Discord and Telegram accounts without exposing sensitive information or allowing unauthorized access.

## Security Architecture

### Key Security Principles

1. **Discord-Only Code Issuance**: Only the official Discord bot can issue linking codes
2. **API Key Protection**: Code issuance requires a secure API key
3. **Single-Use Codes**: Each code can only be used once
4. **Time-Limited Codes**: Codes expire after 10 minutes
5. **No Public Endpoints**: Code verification is the only public endpoint

### Flow Diagram

```
User starts in Discord:
1. User runs /link-telegram
2. Discord bot calls /api/telegrams/issue-code (with API key)
3. Backend generates unique code and returns it
4. Discord bot sends code to user via DM

User completes in Telegram:
1. User runs /verify <code>
2. Telegram bot calls /api/telegrams/verify-code
3. Backend verifies code and links accounts
4. Success message sent to user
```

## API Endpoints

### POST /api/telegrams/issue-code
**Security**: Requires API key (Discord bot only)

Issues a linking code for a Discord user.

**Request:**
```json
{
  "discord_id": "123456789012345678",
  "telegram_id": "optional_pre_binding"
}
```

**Response:**
```json
{
  "success": true,
  "code": "a1b2c3"
}
```

### POST /api/telegrams/verify-code
**Security**: Public endpoint (Telegram bot)

Verifies a linking code and creates the account link.

**Request:**
```json
{
  "code": "a1b2c3",
  "telegram_id": "987654321"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Account successfully linked"
}
```

### GET /api/telegrams/linked-accounts
**Security**: Requires API key (Admin only)

Returns all linked accounts for debugging/admin purposes.

**Response:**
```json
[
  {
    "discord_id": "123456789012345678",
    "telegram_id": "987654321",
    "linked_at": "2024-01-01T12:00:00Z"
  }
]
```

## Discord Bot Commands

### /link-telegram
Generates a linking code and sends it to the user via DM.

**Usage:**
```
/link-telegram
```

**Response:** Code sent via DM with instructions.

### /verify <code>
Legacy command for verification (existing functionality).

## Telegram Bot Commands

### /verify [code]
Verifies a linking code or shows instructions.

**Usage:**
```
/verify a1b2c3
```

**Response:** Success message or error details.

### /verify (no code)
Shows linking instructions.

**Response:** Step-by-step instructions for linking accounts.

## Configuration

### Environment Variables

```bash
# API Configuration
API_KEY=your_secure_api_key_here
API_ADDR=:8080
API_URL=http://localhost:8080

# Discord Bot Configuration
DISCORD_BOT_TOKEN=your_discord_bot_token_here
DISCORD_GUILD_ID=your_discord_guild_id_here
DISCORD_ROLE_ID=your_discord_role_id_here

# Telegram Bot Configuration
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here
TELEGRAM_CHAT_ID=your_telegram_chat_id_here
```

### Running the Server

```bash
go run cmd/server/main.go \
  --api-key=your_secure_api_key \
  --discord-token=your_discord_bot_token \
  --discord-guild=your_guild_id \
  --telegram-token=your_telegram_bot_token
```

## Security Considerations

### API Key Management
- Use a strong, randomly generated API key
- Store the API key securely (environment variables, secret management)
- Rotate the API key periodically
- Never expose the API key in client-side code

### Code Generation
- Codes are 6-character hexadecimal strings
- Generated using cryptographically secure random number generator
- 3 bytes of entropy = 16,777,216 possible codes

### Rate Limiting
Consider implementing rate limiting for:
- Code issuance (per Discord user)
- Code verification attempts (per Telegram user)
- API endpoints (per IP address)

### Database Security
- Store codes and account links securely
- Implement proper access controls
- Consider encryption for sensitive data
- Regular cleanup of expired codes

## Error Handling

### Common Error Scenarios

1. **Invalid Code**: Code doesn't exist or has expired
2. **Already Used**: Code has already been consumed
3. **Account Already Linked**: Discord or Telegram account already linked
4. **API Key Missing**: Missing or invalid API key for protected endpoints
5. **Network Issues**: Connection problems between bots and API

### Error Responses

All API endpoints return consistent error responses:

```json
{
  "success": false,
  "error": "Descriptive error message"
}
```

## Monitoring and Logging

### Key Metrics to Monitor
- Code issuance rate
- Code verification success rate
- Failed verification attempts
- API response times
- Error rates by endpoint

### Log Events
- Code issuance (with Discord ID)
- Code verification attempts (with Telegram ID)
- Successful account links
- Failed verification attempts
- API key validation failures

## Development and Testing

### Local Development
1. Set up environment variables
2. Run the server: `go run cmd/server/main.go`
3. Test with Discord and Telegram bots
4. Use the admin endpoint to verify links

### Testing Checklist
- [ ] Code issuance works with valid API key
- [ ] Code issuance fails without API key
- [ ] Code verification works with valid code
- [ ] Code verification fails with invalid code
- [ ] Code verification fails with expired code
- [ ] Code verification fails with already used code
- [ ] Account linking prevents duplicate links
- [ ] Expired codes are cleaned up automatically

## Troubleshooting

### Common Issues

1. **Discord bot can't send DMs**
   - Check user privacy settings
   - Verify bot has proper permissions

2. **Telegram bot not responding**
   - Verify bot token is correct
   - Check bot is not blocked by user

3. **API connection issues**
   - Verify API URL is correct
   - Check network connectivity
   - Verify API key is valid

4. **Codes not working**
   - Check code hasn't expired (10 minutes)
   - Verify code hasn't been used already
   - Check for typos in code entry

### Debug Commands

Use the admin endpoint to check linked accounts:
```bash
curl -H "Authorization: Bearer YOUR_API_KEY" \
  http://localhost:8080/api/telegrams/linked-accounts
```

## Future Enhancements

### Potential Improvements
1. **Database Persistence**: Store data in PostgreSQL/Redis instead of memory
2. **Rate Limiting**: Implement proper rate limiting
3. **Audit Logging**: Detailed audit trail of all operations
4. **Webhook Support**: Real-time notifications for account links
5. **Multi-Platform Support**: Extend to other platforms (Slack, etc.)
6. **Admin Dashboard**: Web interface for managing links
7. **Bulk Operations**: Support for bulk account linking
8. **Analytics**: Detailed analytics and reporting

### Security Enhancements
1. **Code Encryption**: Encrypt codes in storage
2. **IP Whitelisting**: Restrict API access by IP
3. **JWT Tokens**: Use JWT for API authentication
4. **2FA Support**: Add two-factor authentication
5. **Session Management**: Proper session handling 