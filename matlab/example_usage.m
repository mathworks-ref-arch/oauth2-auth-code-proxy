%% OAuth2 Token Proxy - MATLAB Client Example
% Demonstrates the authorization code flow with PKCE via the oauth2-auth-code-proxy.

baseUrl = "http://localhost/v1";

%% 1. Health check — verify the proxy is running
aliveResp = webread(baseUrl + "/admin/alive");
fprintf("Proxy version: %s\n", aliveResp.version);

%% 2. List available apps
appsResp = webread(baseUrl + "/apps");
disp("Configured apps:");
for i = 1:numel(appsResp.apps)
    fprintf("  [%d] %s — %s\n", i, appsResp.apps(i).id, appsResp.apps(i).description);
end

%% 3. Start authorization (generates PKCE challenge + auth URL)
appId = appsResp.apps(1).id;  % pick the first app, or set manually

authorizeBody = struct( ...
    'app_id', appId, ...
    'redirect_url', "http://localhost:31000/callback" ...
    );

opts = weboptions('MediaType', 'application/json', 'ContentType', 'json');
authResp = webwrite(baseUrl + "/authorize", authorizeBody, opts);

% Save the code_verifier — you need it for the token exchange
codeVerifier = authResp.code_verifier;
authorizationUrl = authResp.authorization_url;

fprintf("Open this URL in your browser to log in:\n%s\n", authorizationUrl);

%% 4. Open browser for user login
web(authorizationUrl, '-browser');

%% 5. Listen for the OAuth2 callback on localhost:31000
callbackPort = 31000;
server = tcpserver("0.0.0.0", callbackPort);
fprintf("Waiting for OAuth2 callback on port %d...\n", callbackPort);
fprintf("If the browser shows a certificate error, paste the code manually.\n");

% Wait for the browser redirect (timeout after 120 seconds)
timedOut = false;
tic;
while server.NumBytesAvailable == 0
    if toc > 120
        timedOut = true;
        break;
    end
    pause(0.1);
end

if timedOut
    delete(server);
    fprintf("\nListener timed out — the redirect page likely failed to load (TLS cert issue).\n");
    fprintf("Copy the full URL from the browser address bar that starts with:\n");
    fprintf("  https://localhost/v1/redirect?code=...&state=...\n\n");
    callbackUrl = input("Paste the full URL here: ", 's');
    parsedUrl = matlab.net.URI(callbackUrl);
    queryParams = parsedUrl.Query;
    authCode = "";
    authState = "";
    authScope = "";
    authError = "";
    authErrorDesc = "";
    for i = 1:numel(queryParams)
        if queryParams(i).Name == "code"
            authCode = string(queryParams(i).Value);
        elseif queryParams(i).Name == "state"
            authState = string(queryParams(i).Value);
        elseif queryParams(i).Name == "scope"
            authScope = string(queryParams(i).Value);
        elseif queryParams(i).Name == "error"
            authError = string(queryParams(i).Value);
        elseif queryParams(i).Name == "error_description"
            authErrorDesc = urldecode(string(queryParams(i).Value));
        end
    end
else
    % Read the HTTP request
    pause(0.5);
    rawRequest = read(server, server.NumBytesAvailable, "string");

    % Send a minimal HTTP response so the browser tab can be closed
    httpResponse = "HTTP/1.1 200 OK" + newline + ...
        "Content-Type: text/html" + newline + ...
        "Connection: close" + newline + newline + ...
        "<html><body><h3>You may close this tab.</h3></body></html>";
    write(server, uint8(char(httpResponse)));
    pause(0.2);
    delete(server);

    % Extract the authorization code, state, and scope from the query string
    requestLine = extractBefore(rawRequest, newline);
    urlPath = extractBetween(requestLine, "GET ", " HTTP");
    parsedUrl = matlab.net.URI(urlPath);
    queryParams = parsedUrl.Query;
    authCode = "";
    authState = "";
    authScope = "";
    authError = "";
    authErrorDesc = "";
    for i = 1:numel(queryParams)
        if queryParams(i).Name == "code"
            authCode = string(queryParams(i).Value);
        elseif queryParams(i).Name == "state"
            authState = string(queryParams(i).Value);
        elseif queryParams(i).Name == "scope"
            authScope = string(queryParams(i).Value);
        elseif queryParams(i).Name == "error"
            authError = string(queryParams(i).Value);
        elseif queryParams(i).Name == "error_description"
            authErrorDesc = urldecode(string(queryParams(i).Value));
        end
    end
end

% The proxy forwards provider errors (e.g. access_denied when the user
% cancels consent) instead of a code
if authError == "access_denied"
    error("Sign-in was cancelled or consent was not granted. %s", authErrorDesc);
elseif authError ~= ""
    error("Authorization failed (%s). %s", authError, authErrorDesc);
end
if authCode == ""
    error("No authorization code found in callback.");
end
fprintf("Received authorization code.\n");

% URL-decode scope (the callback passes it percent-encoded)
authScope = urldecode(authScope);

%% 6. Exchange the authorization code for tokens
tokenBody = struct( ...
    'app_id', appId, ...
    'grant_type', "authorization_code", ...
    'code', authCode, ...
    'state', authState, ...
    'scope', authScope, ...
    'code_verifier', codeVerifier ...
    );

tokenResp = webwrite(baseUrl + "/token", tokenBody, opts);

%% 7. Read from Resource

%Replace the below endpoint with the endpoint of your resource.
url = "https://<path_to_resources>";

opts = weboptions( ...
    'HeaderFields', ["Authorization", "Bearer " + tokenResp.access_token], ...
    'ContentType', 'json');
data = webread(sheetsUrl, sheetsOpts);

%% 8. Refresh the token when it expires
if isfield(tokenResp, 'refresh_token') && ~isempty(tokenResp.refresh_token)
    refreshBody = struct( ...
        'app_id', appId, ...
        'grant_type', "refresh_token", ...
        'refresh_token', tokenResp.refresh_token ...
        );

    refreshResp = webwrite(baseUrl + "/token", refreshBody, opts);
end