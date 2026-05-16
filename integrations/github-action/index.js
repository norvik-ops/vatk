const core = require('@actions/core');
const https = require('https');
const http = require('http');
const url = require('url');

/**
 * Fetches all secrets for the given project+environment from ShieldStack
 * and injects them as masked environment variables.
 */
async function run() {
  const baseUrl = core.getInput('url').replace(/\/$/, '');
  const token = core.getInput('token');
  const project = core.getInput('project');
  const env = core.getInput('environment');

  core.info(`Fetching secrets from ${baseUrl} for project=${project} env=${env}`);

  // 1. Resolve project ID (if name given, look it up; if UUID, use directly)
  let projectId = project;
  if (!isUUID(project)) {
    const projects = await apiGet(baseUrl, token, '/api/v1/secretops/projects');
    const found = projects.find(p => p.name === project || p.slug === project);
    if (!found) {
      core.setFailed(`Project "${project}" not found`);
      return;
    }
    projectId = found.id;
  }

  // 2. Resolve environment ID
  const envs = await apiGet(baseUrl, token, `/api/v1/secretops/projects/${projectId}/envs`);
  const foundEnv = envs.find(e => e.name === env);
  if (!foundEnv) {
    core.setFailed(`Environment "${env}" not found in project ${projectId}`);
    return;
  }
  const envId = foundEnv.id;

  // 3. List secret keys
  const secretKeys = await apiGet(baseUrl, token, `/api/v1/secretops/projects/${projectId}/envs/${envId}/secrets`);

  // 4. Fetch each secret value, mask it, and export as env var
  for (const sk of secretKeys) {
    const secret = await apiGet(baseUrl, token, `/api/v1/secretops/projects/${projectId}/envs/${envId}/secrets/${sk.key}`);
    const value = secret.value;

    // Mask the value so it never appears in logs
    core.setSecret(value);

    // Export as environment variable for subsequent steps
    core.exportVariable(sk.key, value);
    core.info(`Injected secret: ${sk.key}`);
  }

  core.info(`Done — injected ${secretKeys.length} secret(s)`);
}

/** Simple promise-based HTTPS/HTTP GET that returns parsed JSON. */
function apiGet(baseUrl, token, path) {
  return new Promise((resolve, reject) => {
    const parsed = new url.URL(baseUrl + path);
    const lib = parsed.protocol === 'https:' ? https : http;

    const options = {
      hostname: parsed.hostname,
      port: parsed.port || (parsed.protocol === 'https:' ? 443 : 80),
      path: parsed.pathname + parsed.search,
      method: 'GET',
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: 'application/json',
      },
    };

    const req = lib.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => { data += chunk; });
      res.on('end', () => {
        if (res.statusCode < 200 || res.statusCode >= 300) {
          reject(new Error(`HTTP ${res.statusCode} for ${path}: ${data}`));
          return;
        }
        try {
          resolve(JSON.parse(data));
        } catch (e) {
          reject(new Error(`Failed to parse JSON from ${path}: ${e.message}`));
        }
      });
    });

    req.on('error', reject);
    req.end();
  });
}

/** Returns true when the string looks like a UUID v4. */
function isUUID(s) {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(s);
}

run().catch(core.setFailed);
