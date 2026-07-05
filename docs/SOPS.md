# SOPS / AGE: keys, encrypt, decrypt

DD-UI integrates with **SOPS** to keep secrets encrypted at rest in your IaC repo. The backend calls the `sops` CLI; ensure it’s on `PATH` (our image installs it to `/usr/local/bin/sops`).

### Generate AGE key pair (server-side decrypt capability)
On a **secure workstation** or secrets box:
```bash
# Generate a private key; prints the public recipient on stderr
age-keygen -o /opt/docker/dd-ui/secrets/sops_age_key.txt
# Show (or copy) the public recipient for encryption (starts with "age1")
age-keygen -y /opt/docker/dd-ui/secrets/sops_age_key.txt
```
Wire it into Compose as a Docker secret (see `sops_age_key` in the compose file above), and point DD-UI at it:
```
SOPS_AGE_KEY_FILE=/run/secrets/sops_age_key
```
> **Never** commit the private key to Git. Treat `/opt/docker/dd-ui/secrets/sops_age_key.txt` like any other production secret.

### Choose encrypt recipients
To encrypt, SOPS needs one or more **AGE recipients** (public keys). You have two main options:

1. **Environment variable (no repo config required)**
   Set `SOPS_AGE_RECIPIENTS` with one or more recipients (space-separated):
   ```
   SOPS_AGE_RECIPIENTS="age1teamUser1... age1teamUser2... age1ciKey..."
   ```
   DD-UI will pass each recipient to `sops` as `--age <recipient>` during encryption.

2. **`.sops.yaml` in your repo**
   Store creation rules in the repo so `sops` knows what to use per path:
   ```yaml
   # /data/.sops.yaml
   creation_rules:
     - path_regex: 'docker-compose/.+/.+/.+\.env$'
       encrypted_regex: '^(SECRET_|PASSWORD_|API_KEY|TOKEN)'
       key_groups:
         - age:
             - age1teamUser1...
             - age1ciKey...
       # (Optional) tell sops the input format for .env files
       # (DD-UI already hints this when encrypting *.env)
       # unencrypted_suffix: _unencrypted
   ```

> If you see `sops: encrypt failed: ... config file not found, or has no creation rules, and no keys provided ...`, it means neither `SOPS_AGE_RECIPIENTS` nor a `.sops.yaml` with matching creation rules were found. Provide recipients or add a config.

### Encrypting files
- **From the DD-UI UI**: creating/updating a file with the **SOPS** toggle ON (or naming it `*_private.env` / `*_secret.env`) will attempt to run:
  - `sops -e -i [--input-type dotenv] <file>`
  - Plus `--age <recipient>` for each recipient present in `SOPS_AGE_RECIPIENTS`.
- **From CLI** (local dev):
  ```bash
  # .env files (dotenv-aware):
  sops --input-type dotenv --age age1recipient... -e -i /data/docker-compose/host/stack/app.env
  # generic YAML/JSON/TOML:
  sops --age age1recipient... -e -i /data/docker-compose/host/stack/compose.yaml
  ```

### Decrypting (gated reveal)
- Decryption in DD-UI is **explicitly gated**:
  - Server-side must allow it: `DD_UI_ALLOW_SOPS_DECRYPT=true`
  - UI sends a confirmation header: `X-Confirm-Reveal: yes`
  - Backend calls: `sops -d <file>` and returns the plaintext (not persisted).
- If decryption is not allowed you’ll see `403 Forbidden: decrypt disabled on server`.
- If SOPS fails, the backend returns the combined stderr/stdout so you can see the exact `sops` error.

**Security notes**
- DD-UI never stores plaintext on disk—decrypt results stream back to the client only on explicit user action.
- Consider running the backend on a host you already trust with decryption keys, and restrict who can log in.
