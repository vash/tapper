# Tapper - Terraform Profile Manager

A CLI tool that simplifies running Terraform commands with different backend configurations and variable files. Tapper automatically detects profiles from matching `.tfbackend` and `.tfvars` files and supports parallel execution across multiple profiles with real-time streaming output.

## ✨ Features

- **Auto-detection** - Automatically discovers profiles from file system structure
- **Parallel execution** - Run terraform commands across multiple profiles simultaneously
- **Real-time streaming** - See output from all profiles in real-time with color coding
- **Interactive selection** - Choose profiles with fuzzy search (fzf) or fallback menu
- **Workspace isolation** - Each profile runs in isolated temporary workspace
- **AWS SSO integration** - Automatic SSO token refresh when expired
- **Plan approval** - Review terraform plans before execution

## 🚀 Installation

### Download from releases
```bash
# Download latest release for your platform
curl -L https://github.com/yourusername/tapper/releases/latest/download/tapper_linux_amd64.tar.gz | tar xz
sudo mv tapper /usr/local/bin/
```

### Build from source
```bash
git clone https://github.com/yourusername/tapper.git
cd tapper
go build -o bin/tapper ./cmd/tapper
```

## 📁 Directory Structure

Tapper expects your Terraform project to follow this structure:

```
your-terraform-project/
├── main.tf                    # Your terraform configuration
├── backend/                   # Backend configuration files
│   ├── dev.tfbackend
│   ├── staging.tfbackend
│   └── prod.tfbackend
└── vars/                      # Variable files
    ├── dev.tfvars
    ├── staging.tfvars
    └── prod.tfvars
```

**Profile Detection:** Tapper automatically creates profiles by matching filenames:
- `backend/dev.tfbackend` + `vars/dev.tfvars` = `dev` profile
- `backend/prod.tfbackend` + `vars/prod.tfvars` = `prod` profile

## 🎯 Usage

### Run terraform plan
```bash
# Interactive profile selection
tapper plan

# Plan specific profile
tapper plan dev

# Plan multiple profiles in parallel
tapper plan dev staging prod
```

### Run terraform apply
```bash
# Interactive selection with plan approval
tapper apply

# Apply to specific profile
tapper apply prod
```

### Run terraform destroy
```bash
# Interactive selection
tapper destroy

# Destroy specific profile
tapper destroy dev
```

### Manage profiles
```bash
# List all detected profiles
tapper profile list

# Get help for profile management
tapper profile --help
```

## 🔧 Requirements

- **Go 1.23.3+** (for building from source)
- **Terraform** - Must be available in PATH
- **fzf** (optional) - For enhanced interactive selection. Falls back to simple menu if not available.

### Installing fzf (optional but recommended)
```bash
# macOS
brew install fzf

# Linux
sudo apt install fzf  # Debian/Ubuntu
sudo dnf install fzf  # Fedora

# Or install via Go
go install github.com/junegunn/fzf@latest
```

## 🌟 Example Workflow

1. **Set up your project structure:**
```bash
mkdir -p backend vars
echo 'bucket = "my-terraform-state-dev"' > backend/dev.tfbackend
echo 'environment = "dev"' > vars/dev.tfvars
```

2. **Plan across profiles:**
```bash
tapper plan
# Interactively select profiles, review plans, approve execution
```

3. **Apply changes:**
```bash
tapper apply dev
# Runs terraform apply with dev profile configuration
```

## 🎨 Features in Detail

### Real-time Streaming Output
- Color-coded output per profile
- Timestamps for all operations
- Clear success/failure indicators

### Workspace Isolation
- Each profile runs in a temporary workspace
- Automatic cleanup after execution
- Prevents state conflicts between profiles

### AWS SSO Integration
- Automatic detection of expired SSO tokens
- Automatic `aws sso login` when needed
- Seamless multi-profile AWS operations

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes: `git commit -m 'Add amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- Uses [fzf](https://github.com/junegunn/fzf) for interactive selection
- Inspired by the need for better Terraform multi-environment workflows 