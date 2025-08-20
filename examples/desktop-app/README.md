# Desktop App Example

A modern desktop application demonstrating external tool authentication patterns with a rich web-based user interface for AWS resource management.

## 🎯 Features

- **Modern Web UI**: Responsive, dark-mode-ready interface with Tailwind CSS
- **Cross-Platform**: Runs on Windows, macOS, and Linux
- **Multiple Authentication**: AWS SSO, profiles, and interactive authentication
- **Real-Time Dashboard**: Live AWS resource monitoring with auto-refresh
- **S3 Browser**: Browse and manage S3 buckets and objects
- **EC2 Management**: Monitor EC2 instances across regions
- **Theme Support**: Light, dark, and auto themes
- **System Tray**: Minimize to system tray (planned)
- **Auto-Launch**: Automatic browser opening

## 🚀 Quick Start

### Build and Run

```bash
cd examples/desktop-app
go build -o aws-desktop-app main.go
./aws-desktop-app
```

The app will:
1. Start a local web server (typically on port 8080)
2. Automatically open your default web browser
3. Present a modern AWS management interface

### First-Time Setup

1. **Launch the Application**
   ```bash
   ./aws-desktop-app
   ```

2. **Configure Authentication**
   - Click "Setup Authentication" on the dashboard
   - Choose your preferred authentication method
   - Follow the guided setup process

3. **Start Managing Resources**
   - View your AWS resources on the dashboard
   - Browse S3 buckets and objects
   - Monitor EC2 instances across regions

## 🖥️ User Interface

### Dashboard
- **Authentication Status**: Real-time auth status with visual indicators
- **Resource Overview**: Quick stats on S3 buckets, EC2 instances, etc.
- **Quick Actions**: Common operations accessible with one click

### S3 Browser
- **Bucket Listing**: View all S3 buckets with creation dates and regions
- **Object Browser**: Navigate bucket contents (planned)
- **Upload/Download**: Drag-and-drop file operations (planned)

### EC2 Management
- **Instance Overview**: All instances across regions
- **Status Monitoring**: Real-time instance states
- **Region Filtering**: Filter instances by AWS region

### Settings
- **Authentication**: Configure AWS authentication methods
- **Appearance**: Theme selection and UI preferences
- **Features**: Enable/disable specific functionality

## 🔧 Authentication Methods

### 1. AWS SSO (Recommended)
```
Setup → AWS SSO → Enter Start URL → Browser Authentication
```

- Most secure for organizations
- Automatic token refresh
- Browser-based authentication flow

### 2. AWS Profile
```
Setup → AWS Profile → Select Existing Profile
```

- Uses existing `~/.aws/credentials`
- Good for development environments
- Quick setup for existing AWS CLI users

### 3. Interactive Authentication
```
Setup → Interactive → Guided Process
```

- Step-by-step authentication guidance
- Suitable for first-time AWS users
- Built-in help and troubleshooting

## ⚙️ Configuration

### Application Configuration

The app stores settings in `~/.aws-desktop-app/config.json`:

```json
{
  "debug": false,
  "theme": "auto",
  "aws_region": "us-east-1",
  "auth": {
    "method": "sso",
    "region": "us-east-1",
    "session_duration": 3600,
    "cache_enabled": true,
    "sso": {
      "start_url": "https://company.awsapps.com/start",
      "region": "us-east-1"
    }
  },
  "ui": {
    "theme": "auto",
    "auto_refresh": true,
    "refresh_interval": 30,
    "notifications": true
  },
  "features": {
    "s3_browser": true,
    "ec2_management": true,
    "logs_viewer": true,
    "system_tray": true
  }
}
```

### Environment Variables

Override settings with environment variables:

```bash
export AWS_DESKTOP_APP_DEBUG=true
export AWS_DESKTOP_APP_THEME=dark
export AWS_DESKTOP_APP_AWS_REGION=us-west-2
```

## 🎨 Themes and Customization

### Built-in Themes
- **Auto**: Follows system preference
- **Light**: Clean, bright interface
- **Dark**: Easy on the eyes, battery-friendly

### Theme Switching
- Use the theme toggle button in the navigation
- Or configure in Settings → Appearance
- Changes apply immediately without restart

### UI Customization
- **Compact Mode**: Reduced spacing and smaller elements
- **Auto Refresh**: Automatic resource updates
- **Notifications**: Toast notifications for operations

## 🔌 API Endpoints

The desktop app exposes a local API for advanced integration:

### Status Endpoints
```
GET /api/status                 # Application status
GET /api/auth/status           # Authentication status
```

### Authentication Endpoints
```
POST /api/auth/setup           # Configure authentication
POST /api/auth/test            # Test authentication
POST /api/auth/clear           # Clear configuration
```

### Resource Endpoints
```
GET /api/s3/buckets           # List S3 buckets
GET /api/ec2/instances        # List EC2 instances
```

### Configuration Endpoints
```
GET /api/config               # Get configuration
POST /api/config              # Update configuration
```

## 🛠️ Development

### Project Structure

```
examples/desktop-app/
├── main.go                    # Application entry point
├── internal/
│   ├── auth/                  # Authentication management
│   │   └── manager.go         # Auth manager with status tracking
│   ├── config/                # Configuration management
│   │   └── config.go          # JSON-based configuration
│   └── ui/                    # Web UI handlers
│       └── handler.go         # HTTP request handlers
└── web/                       # Web interface assets
    ├── static/
    │   ├── css/
    │   │   └── app.css        # Custom styles
    │   └── js/
    │       └── app.js         # Frontend JavaScript
    └── templates/
        └── index.html         # Main HTML template
```

### Building for Different Platforms

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o aws-desktop-app.exe main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o aws-desktop-app-darwin main.go

# Linux
GOOS=linux GOARCH=amd64 go build -o aws-desktop-app-linux main.go
```

### Development Mode

```bash
# Run with debug logging
DEBUG=true go run main.go

# Or set environment variable
export AWS_DESKTOP_APP_DEBUG=true
go run main.go
```

### Adding New Features

1. **Backend**: Add API endpoints in `internal/ui/handler.go`
2. **Frontend**: Add UI components in `web/templates/index.html`
3. **Styling**: Update styles in `web/static/css/app.css`
4. **JavaScript**: Add functionality in `web/static/js/app.js`

## 🔒 Security Features

### Local-Only Access
- Web server binds to `127.0.0.1` only
- No external network access
- Automatic port selection

### Secure Credential Storage
- No long-lived credentials stored
- Temporary tokens with automatic refresh
- Encrypted configuration files

### Authentication Security
- External ID validation for cross-account roles
- Session timeout enforcement
- Automatic credential expiration

## 🚨 Troubleshooting

### Common Issues

#### Port Already in Use
```
Error: Failed to start server on port 8080
```

**Solution**: The app automatically finds available ports (8080-8099). If all are busy, close other applications.

#### Browser Doesn't Open
```
Warning: Failed to open browser
```

**Solution**: Manually navigate to the URL shown in the console output.

#### Authentication Fails
```
Error: Failed to get AWS configuration
```

**Solutions**:
- Check your internet connection
- Verify AWS credentials are valid
- Try refreshing authentication in the UI
- Clear authentication and reconfigure

### Debug Mode

Enable debug logging for troubleshooting:

```bash
# Environment variable
export AWS_DESKTOP_APP_DEBUG=true
./aws-desktop-app

# Or in-app
Settings → Enable Debug Logging
```

### Log Files

Application logs are stored in:
- **macOS/Linux**: `~/.aws-desktop-app/app.log`
- **Windows**: `%USERPROFILE%\.aws-desktop-app\app.log`

## 🚀 Deployment

### Packaging for Distribution

#### macOS App Bundle
```bash
# Create app bundle structure
mkdir -p "AWS Desktop App.app/Contents/MacOS"
mkdir -p "AWS Desktop App.app/Contents/Resources"

# Copy binary
cp aws-desktop-app-darwin "AWS Desktop App.app/Contents/MacOS/aws-desktop-app"

# Create Info.plist
cat > "AWS Desktop App.app/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>aws-desktop-app</string>
    <key>CFBundleIdentifier</key>
    <string>com.example.aws-desktop-app</string>
    <key>CFBundleName</key>
    <string>AWS Desktop App</string>
    <key>CFBundleVersion</key>
    <string>1.0.0</string>
</dict>
</plist>
EOF
```

#### Windows Installer
```bash
# Using NSIS or similar installer generator
# Package aws-desktop-app.exe with installer script
```

#### Linux AppImage
```bash
# Package as AppImage for universal Linux distribution
# Use appimagetool or similar
```

## 🔮 Planned Features

### Version 2.0
- [ ] System tray integration
- [ ] Keyboard shortcuts
- [ ] Multi-tab interface
- [ ] CloudWatch Logs viewer
- [ ] IAM policy helper
- [ ] Cost explorer integration

### Version 3.0
- [ ] Plugin system
- [ ] Custom dashboards
- [ ] Resource templates
- [ ] Automated workflows
- [ ] Team collaboration features

## 📚 API Reference

### JavaScript API

The desktop app provides a JavaScript API for custom integrations:

```javascript
// Check authentication status
const authStatus = await window.awsApp.getAuthStatus();

// Refresh resources
await window.awsApp.refreshResources();

// Get S3 buckets
const buckets = await window.awsApp.getS3Buckets();

// Show notification
window.awsApp.showNotification('Operation completed', 'success');
```

### Theme API

```javascript
// Get current theme
const theme = window.awsApp.getTheme();

// Set theme
window.awsApp.setTheme('dark');

// Listen for theme changes
window.awsApp.onThemeChange((theme) => {
    console.log('Theme changed to:', theme);
});
```

## 🤝 Contributing

Contributions are welcome! Areas for improvement:

1. **UI/UX Enhancements**: Better visual design and user experience
2. **New AWS Services**: Add support for more AWS services
3. **Platform Features**: System tray, notifications, shortcuts
4. **Performance**: Optimize resource loading and caching
5. **Documentation**: Improve user guides and developer docs

## 📄 License

This example is part of the AWS Remote Access Patterns project and follows the same MIT license.

---

This desktop application demonstrates how to create a modern, user-friendly interface for AWS resource management while maintaining the highest security standards with temporary credentials and proper authentication patterns.