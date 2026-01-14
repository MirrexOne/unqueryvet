# Publishing to VS Code Marketplace

This guide explains how to publish the Unqueryvet extension to the VS Code Marketplace.

## Prerequisites

1. **Microsoft Account** - You need a Microsoft account
2. **Azure DevOps Organization** - Free to create at https://dev.azure.com
3. **Publisher Account** - Created in VS Code Marketplace

## Step 1: Create Azure DevOps Organization

1. Go to https://dev.azure.com
2. Sign in with your Microsoft account
3. Create a new organization (or use existing one)

## Step 2: Create Personal Access Token (PAT)

1. In Azure DevOps, click on your profile icon (top right)
2. Select **Personal access tokens**
3. Click **+ New Token**
4. Configure the token:
   - **Name**: `vsce-publish` (or any name)
   - **Organization**: Select your organization or "All accessible organizations"
   - **Expiration**: Choose expiration period
   - **Scopes**: Click "Show all scopes", then select:
     - **Marketplace** → **Manage** (check this)
5. Click **Create**
6. **IMPORTANT**: Copy the token immediately - you won't see it again!

## Step 3: Create Publisher

1. Go to https://marketplace.visualstudio.com/manage
2. Sign in with your Microsoft account
3. Click **Create Publisher**
4. Fill in:
   - **ID**: `unqueryvet` (must match `publisher` in package.json)
   - **Name**: `Unqueryvet`
   - Other fields as desired
5. Click **Create**

## Step 4: Add PAT to GitHub Secrets

1. Go to your GitHub repository
2. Navigate to **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Create secret:
   - **Name**: `VSCE_PAT`
   - **Value**: Paste your Azure DevOps PAT
5. Click **Add secret**

## Step 5: Publish

### Automatic (via GitHub Actions)

Create a new release on GitHub:
1. Go to **Releases** → **Create new release**
2. Tag: `v1.0.0` (or your version)
3. Title: `v1.0.0`
4. Publish release

The workflow will automatically:
- Build the extension
- Publish to VS Code Marketplace
- Attach .vsix to the release

### Manual Publishing

```bash
cd extensions/vscode
npm install
npm run compile

# Login with your PAT
npx @vscode/vsce login unqueryvet

# Publish
npx @vscode/vsce publish
```

Or with PAT directly:
```bash
npx @vscode/vsce publish -p YOUR_PAT_TOKEN
```

## Verification

After publishing:
1. Wait 5-10 minutes for processing
2. Search "unqueryvet" in VS Code Extensions
3. Verify the extension appears and can be installed

## Updating the Extension

1. Update version in `package.json`
2. Update `CHANGELOG.md`
3. Create a new GitHub release with new tag

## Troubleshooting

### "Access Denied" Error
- Ensure PAT has **Marketplace → Manage** scope
- Verify PAT is not expired
- Check publisher ID matches in package.json

### "Publisher Not Found" Error
- Create publisher at https://marketplace.visualstudio.com/manage
- Ensure publisher ID in package.json matches exactly

### Extension Not Appearing
- Wait 5-10 minutes after publishing
- Check https://marketplace.visualstudio.com/manage for status

## Links

- [VS Code Publishing Guide](https://code.visualstudio.com/api/working-with-extensions/publishing-extension)
- [Azure DevOps PAT](https://docs.microsoft.com/en-us/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate)
- [Marketplace Publisher](https://marketplace.visualstudio.com/manage)
