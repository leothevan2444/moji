using System.Net;
using System.Globalization;
using System.Security.Cryptography;
using System.Text;
using Jackett.Common.Indexers;
using Jackett.Common.Indexers.Definitions;
using Jackett.Common.Models;
using Jackett.Common.Models.Config;
using static Jackett.Common.Models.IndexerConfig.ConfigurationData;
using Jackett.Common.Services.Interfaces;
using Jackett.Common.Utils.Clients;
using Jackett.Common.Utils;
using Newtonsoft.Json;
using Newtonsoft.Json.Linq;
using NLog;
using YamlDotNet.Serialization;
using YamlDotNet.Serialization.NamingConventions;
using WebClient = Jackett.Common.Utils.Clients.WebClient;
using WebRequest = Jackett.Common.Utils.Clients.WebRequest;

Encoding.RegisterProvider(CodePagesEncodingProvider.Instance);
var input = JObject.Parse(File.ReadAllText(args[0]));
if (input["relativeDateFilters"] is JArray relativeFilters)
{
    var helper = new JObject { ["engine"] = "jackett", ["mode"] = "relative-date-helper", ["requests"] = new JArray(), ["results"] = new JArray(), ["relativeDates"] = new JArray() };
    try
    {
        if (relativeFilters.Count == 0 || relativeFilters.Count > 100 || input["now"] == null)
            throw new ArgumentException("A fixed clock and 1..100 inputs are required");
        // JObject can already hold a DateTime token. Keep its subsecond ticks
        // instead of converting through invariant display text (which drops them).
        var clock = input.Value<DateTime>("now").ToUniversalTime();
        foreach (var test in relativeFilters.Cast<JObject>())
        {
            var name = test.Value<string>("name");
            var text = test.Value<string>("input");
            if ((name != "timeago" && name != "reltime" && name != "fuzzytime") || text == null || Encoding.UTF8.GetByteCount(text) > 4096)
                throw new ArgumentException("Invalid relative-date helper input");
            string value = "";
            bool invalid = false;
            try
            {
                var parsed = name == "fuzzytime" ? DateTimeUtil.FromUnknown(text, null, clock) : DateTimeUtil.FromTimeAgo(text, clock);
                value = parsed.ToString(DateTimeUtil.Rfc1123ZPattern, CultureInfo.InvariantCulture);
            }
            catch (Exception) { invalid = true; }
            if (invalid != (test.Value<bool?>("expectInvalid") ?? false))
                throw new ArgumentException("Relative-date outcome differs from declared expectation");
            ((JArray)helper["relativeDates"]).Add(new JObject { ["name"] = name, ["input"] = text, ["value"] = value, ["invalid"] = invalid });
        }
    }
    catch (Exception error) { helper["error"] = error.GetType().Name + ": " + error.Message; }
    File.WriteAllText(args[1], helper.ToString(Formatting.Indented));
    return;
}
var client = new ScriptedClient((JArray)input["responses"]);
var result = new JObject { ["engine"] = "jackett", ["requests"] = client.Requests };
try
{
    var definition = new DeserializerBuilder().WithNamingConvention(CamelCaseNamingConvention.Instance)
        .Build().Deserialize<IndexerDefinition>(input.Value<string>("definition"));
    var indexer = new CardigannIndexer(new MemoryConfiguration(), client, LogManager.GetLogger("reference"), null, new NoCache(), definition);
    var configuration = new JArray();
    foreach (var property in ((JObject)input["settings"] ?? new JObject()).Properties())
        configuration.Add(new JObject { ["id"] = property.Name, ["value"] = property.Value });
    indexer.LoadValuesFromJson(configuration);
    if (input.Value<bool>("authenticate"))
    {
        if (input["challengeAnswer"] != null)
        {
            var setup = await indexer.GetConfigurationForSetup(false);
            var captcha = definition.Login?.Captcha?.Type;
            var challenge = new JObject { ["type"] = captcha, ["prompt"] = "", ["imageSha256"] = "" };
            if (captcha == "text")
            {
                challenge["prompt"] = ((DisplayInfoConfigurationItem)setup.GetDynamic("CaptchaChallenge")).Value;
                ((StringConfigurationItem)setup.GetDynamic("CaptchaAnswer")).Value = input.Value<string>("challengeAnswer");
            }
            else if (captcha == "image")
            {
                var image = ((DisplayImageConfigurationItem)setup.GetDynamic("CaptchaImage")).Value;
                challenge["imageSha256"] = Convert.ToHexString(SHA256.HashData(image)).ToLowerInvariant();
                ((StringConfigurationItem)setup.GetDynamic("CaptchaText")).Value = input.Value<string>("challengeAnswer");
            }
            else throw new ArgumentException("Scripted answer requires a supported login challenge");
            result["challenge"] = challenge;
        }
        await indexer.ApplyConfiguration(configuration);
    }

    var q = (JObject)input["query"] ?? new JObject();
    var query = new TorznabQuery {
        QueryType = q.Value<string>("mode") ?? "search", SearchTerm = q.Value<string>("keywords") ?? "",
        Categories = q["categories"]?.ToObject<int[]>() ?? Array.Empty<int>(),
        Limit = q.Value<int?>("limit") ?? 0, Offset = q.Value<int?>("offset") ?? 0, Cache = false
    };
    foreach (var parameter in ((JObject)q["parameters"] ?? new JObject()).Properties())
    {
        var name = parameter.Name switch { "Ep" => "Episode", "TVRageID" => "RageID", _ => parameter.Name };
        var property = typeof(TorznabQuery).GetProperties().FirstOrDefault(p => string.Equals(p.Name, name, StringComparison.OrdinalIgnoreCase));
        if (property == null || !property.CanWrite) throw new ArgumentException($"Unsupported query parameter {parameter.Name}");
        property.SetValue(query, parameter.Value.ToObject(property.PropertyType));
    }
    var response = await indexer.ResultsForQuery(query, false);
    var releases = response.Releases.ToList();
    result["results"] = JArray.FromObject(releases.Select(r => new {
        title = r.Title, downloadUrl = r.Link?.ToString() ?? "", magnetUri = ApiMagnetText(r.MagnetUri),
        infoHash = r.InfoHash ?? "", detailsUrl = r.Details?.ToString() ?? "", description = r.Description ?? "",
        posterUrl = r.Poster?.ToString() ?? "", size = r.Size, seeders = r.Seeders,
        leechers = r.Peers.HasValue && r.Seeders.HasValue ? r.Peers - r.Seeders : null,
        categories = r.Category.Distinct().OrderBy(id => id).ToArray(),
        publishedAt = r.PublishDate == DateTime.MinValue ? null : r.PublishDate.ToUniversalTime().ToString("O")
    }));
    if (input.Value<bool>("fetch") && releases.Count > 0)
    {
        var first = releases[0];
        if (first.Link != null)
        {
            var bytes = await indexer.Download(first.Link);
            // BaseIndexer returns UTF-8 magnet bytes for both selected magnet
            // links and HTTP redirects. Expose the same artifact union as Moji;
            // preserve the exact text rather than hashing it as torrent bytes.
            var magnet = Encoding.UTF8.GetString(bytes);
            result["artifact"] = magnet.StartsWith("magnet:", StringComparison.Ordinal)
                ? new JObject { ["sha256"] = "", ["magnet"] = magnet }
                : new JObject { ["sha256"] = Convert.ToHexString(SHA256.HashData(bytes)).ToLowerInvariant(), ["magnet"] = "" };
        }
        else if (first.MagnetUri != null)
            result["artifact"] = new JObject { ["sha256"] = "", ["magnet"] = ApiMagnetText(first.MagnetUri) };
    }
}
catch (Exception error)
{
    result["error"] = error.GetType().Name + ": " + error.Message;
}
File.WriteAllText(args[1], result.ToString(Formatting.Indented));

// Jackett exposes a typed Uri through Newtonsoft.Json. Uri.ToString() is a
// display form which can unescape Unicode; use the actual API serialization
// instead. Neither the interpreter nor the Python comparison rewrites the URI.
static string ApiMagnetText(Uri value) => value == null ? ""
    : JsonConvert.DeserializeObject<string>(JsonConvert.SerializeObject(value));

sealed class ScriptedClient : WebClient
{
    readonly JArray responses;
    public readonly JArray Requests = new();
    public ScriptedClient(JArray responses) : base(null, LogManager.GetLogger("http-reference"), null, new ServerConfig(null)) {
        this.responses = responses;
    }
    protected override async Task<WebResult> Run(WebRequest request)
    {
        if (Requests.Count >= 100) throw new InvalidOperationException("Request budget exceeded");
        byte[] body = Array.Empty<byte>();
        string contentType = null;
        if (!string.IsNullOrEmpty(request.RawBody)) body = Encoding.UTF8.GetBytes(request.RawBody);
        else if (request.Type == RequestType.POST) {
            using var content = FormUrlEncodedContentWithEncoding(request.PostData, request.Encoding);
            body = await content.ReadAsByteArrayAsync();
            contentType = content.Headers.ContentType?.ToString();
        }
        var headers = JObject.FromObject(request.Headers);
        if (contentType != null) headers["Content-Type"] = contentType;
        if (!string.IsNullOrEmpty(request.Cookies)) headers["Cookie"] = request.Cookies;
        if (!string.IsNullOrEmpty(request.Referer)) headers["Referer"] = request.Referer;
        Requests.Add(new JObject { ["method"] = request.Type.ToString(), ["url"] = request.Url,
            ["body"] = Convert.ToBase64String(body), ["headers"] = headers });
        var path = new Uri(request.Url).AbsolutePath;
        var fixture = responses.Cast<JObject>().FirstOrDefault(r => r.Value<string>("method") == request.Type.ToString() && r.Value<string>("path") == path)
            ?? throw new InvalidOperationException($"Unscripted request {request.Type} {path}");
        var response = new WebResult { Status = (HttpStatusCode)fixture.Value<int>("status"),
            ContentBytes = Convert.FromBase64String(fixture.Value<string>("body") ?? ""), Request = request };
        foreach (var header in ((JObject)fixture["headers"] ?? new JObject()).Properties())
            response.Headers[header.Name] = header.Value.ToObject<string[]>();
        if (response.Headers.TryGetValue("Location", out var locations)) response.RedirectingTo = locations[0];
        if (response.Headers.TryGetValue("Set-Cookie", out var cookies)) response.Cookies = string.Join("; ", cookies.Select(c => c.Split(';')[0]));
        return response;
    }
}

sealed class MemoryConfiguration : IIndexerConfigurationService
{
    public void Load(IIndexer indexer) { }
    public void Save(IIndexer indexer, JToken configuration) { }
    public void Delete(IIndexer indexer) { }
    public string GetIndexerConfigFilePath(string id) => "/unused";
}

sealed class NoCache : ICacheService
{
    public TimeSpan CacheTTL => TimeSpan.Zero;
    public List<ReleaseInfo> Search(IIndexer indexer, TorznabQuery query) => null;
    public void CacheResults(IIndexer indexer, TorznabQuery query, List<ReleaseInfo> releases) { }
    public IReadOnlyList<TrackerCacheResult> GetCachedResults() => Array.Empty<TrackerCacheResult>();
    public void CleanIndexerCache(IIndexer indexer) { }
    public void CleanCache() { }
}
