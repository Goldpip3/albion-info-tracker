using AlbionInfoTracker.Models;
using Serilog;
using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Net.Http;
using System.Text.Json;
using System.Threading.Tasks;

namespace AlbionInfoTracker.Services;

public class ItemController
{
    private static readonly string CacheDir = Path.Combine(
        Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
        "AlbionInfoTracker");

    private static readonly string CacheFile = Path.Combine(CacheDir, "items.json");

    private const string ItemsUrl = "https://raw.githubusercontent.com/ao-data/ao-bin-dumps/master/formatted/items.json";

    private List<ItemData> _items = new();
    private Dictionary<int, string> _indexToUniqueName = new();

    public bool IsLoaded => _items.Count > 0;

    public async Task LoadItemsAsync()
    {
        try
        {
            Directory.CreateDirectory(CacheDir);

            // Use cached file if less than 24 hours old
            if (File.Exists(CacheFile))
            {
                var fileInfo = new FileInfo(CacheFile);
                if (fileInfo.LastWriteTimeUtc > DateTime.UtcNow.AddHours(-24))
                {
                    await LoadFromFileAsync(CacheFile);
                    if (_items.Count > 0)
                    {
                        Log.Information("Loaded {Count} items from cache", _items.Count);
                        return;
                    }
                }
            }

            // Download fresh data
            using var http = new HttpClient();
            http.Timeout = TimeSpan.FromSeconds(30);
            var json = await http.GetStringAsync(ItemsUrl);
            await File.WriteAllTextAsync(CacheFile, json);
            ParseJson(json);
            Log.Information("Downloaded and cached {Count} items", _items.Count);
        }
        catch (Exception ex)
        {
            Log.Warning(ex, "Failed to load item data, equipment icons will be unavailable");

            // Try loading stale cache as fallback
            if (File.Exists(CacheFile) && _items.Count == 0)
            {
                try
                {
                    await LoadFromFileAsync(CacheFile);
                    Log.Information("Loaded {Count} items from stale cache", _items.Count);
                }
                catch { }
            }
        }
    }

    public string? GetUniqueNameByIndex(int index)
    {
        return _indexToUniqueName.TryGetValue(index, out var name) ? name : null;
    }

    private async Task LoadFromFileAsync(string path)
    {
        var json = await File.ReadAllTextAsync(path);
        ParseJson(json);
    }

    private void ParseJson(string json)
    {
        var items = JsonSerializer.Deserialize<List<ItemData>>(json);
        if (items != null)
        {
            _items = items;
            _indexToUniqueName = items
                .Where(i => !string.IsNullOrEmpty(i.UniqueName))
                .GroupBy(i => i.Index)
                .ToDictionary(g => g.Key, g => g.First().UniqueName);
        }
    }
}
