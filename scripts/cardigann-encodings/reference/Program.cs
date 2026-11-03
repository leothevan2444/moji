using System.Security.Cryptography;
using System.Text;
using System.Text.Json;

Encoding.RegisterProvider(CodePagesEncodingProvider.Instance);
var ids = JsonSerializer.Deserialize<int[]>(File.ReadAllText(args[0]));
var bmp = new string(Enumerable.Range(0, 65536).Select(i => (char)i).ToArray());
var bytes = Enumerable.Range(0, 256).Select(i => (byte)i).ToArray();
var report = new SortedDictionary<int, object>();
foreach (int id in ids)
{
    var encoding = Encoding.GetEncoding(id);
    var encoded = encoding.GetBytes(bmp);
    var decoded = encoding.GetString(bytes);
    report[id] = new {
        name = encoding.WebName,
        encoderBytes = encoded.Length,
        encoderSha256 = Convert.ToHexString(SHA256.HashData(encoded)).ToLowerInvariant(),
        decoderCharacters = decoded.Length,
        decoderSha256 = Convert.ToHexString(SHA256.HashData(Encoding.UTF8.GetBytes(decoded))).ToLowerInvariant(),
        samples = new[] { "ē", "ＡＢ１２", "é", "Ж", "汉", "😀", "ไทย" }.Select(text => new {
            text, hex = Convert.ToHexString(encoding.GetBytes(text)).ToLowerInvariant()
        })
    };
}
File.WriteAllText(args[1], JsonSerializer.Serialize(new {
    runtime = Environment.Version.ToString(), codePages = report
}, new JsonSerializerOptions { WriteIndented = true }));
