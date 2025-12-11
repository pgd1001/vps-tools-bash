# Custom Scripts Directory

Place your custom VPS Tools scripts here.

## Adding a Custom Script

1. Create your script in this directory:
   ```bash
   cat > /opt/vps-tools/custom/my-script.sh << 'EOF'
   #!/bin/bash
   set -euo pipefail
   echo "My custom script"
   EOF
   chmod +x /opt/vps-tools/custom/my-script.sh
   ```

2. Register it in `/etc/vps-tools/plugins.conf`:
   ```
   my-script:custom/my-script.sh:My custom script:custom:true
   ```

3. Use it:
   ```bash
   vps-tools my-script
   ```

## Script Requirements

- Must be executable (`chmod +x`)
- Should use `#!/bin/bash` shebang
- Recommended: use `set -euo pipefail` for safety

## Notes

- This directory is excluded from git updates
- Your scripts persist across VPS Tools updates
- Use the same logging conventions as core scripts for consistency
