using System;
using System.Collections;
using System.Collections.Generic;
using System.Globalization;
using System.Linq;

namespace AlbionInfoTracker.Models;

public static class Extensions
{
    public static Guid? ObjectToGuid(this object value)
    {
        try
        {
            if (value is IEnumerable valueEnumerable)
            {
                var myBytes = valueEnumerable.OfType<byte>().ToArray();
                return new Guid(myBytes);
            }
        }
        catch { }
        return null;
    }

    public static long? ObjectToLong(this object value)
        => value as byte? ?? value as short? ?? value as int? ?? value as long?;

    public static int ObjectToInt(this object value)
        => value as byte? ?? value as short? ?? value as int? ?? 0;

    public static short ObjectToShort(this object value)
        => value as byte? ?? value as short? ?? 0;

    public static byte ObjectToByte(this object value)
        => value as byte? ?? 0;

    public static bool ObjectToBool(this object value)
        => value as bool? ?? false;

    public static double ObjectToDouble(this object value)
        => value as float? ?? value as double? ?? 0;

    public static ulong? ObjectToUlong(this object value)
        => value as byte? ?? value as ushort? ?? value as uint? ?? value as ulong?;

    public static Dictionary<int, T> ToDictionary<T>(this IEnumerable<T> array)
        => array.Select((v, i) => new { Key = i, Value = v }).ToDictionary(o => o.Key, o => o.Value);

    public static Dictionary<int, T> ToDictionary<T>(this T[] array)
    {
        if (array == null) return new Dictionary<int, T>();
        var dict = new Dictionary<int, T>();
        for (int i = 0; i < array.Length; i++) dict[i] = array[i];
        return dict;
    }

    public static double ToPositive(this double value) => value > 0 ? value : -value;

    public static double ToPositiveFromNegativeOrZero(this double healthChange)
        => healthChange >= 0d ? 0d : healthChange.ToPositive();

    public static string ToTimerString(this TimeSpan span)
        => $"{span.Hours:00}:{span.Minutes:00}:{span.Seconds:00}";

    public static string ToShortNumberString(this object num)
    {
        if (num is long l) return GetShortNumber(l);
        if (num is int i) return GetShortNumber(i);
        if (num is not double d) return "0";
        if (double.IsNaN(d)) return "0";
        if (double.IsInfinity(d)) return "MAX";
        return GetShortNumber((decimal)d);
    }

    private static string GetShortNumber(decimal num, CultureInfo? culture = null)
    {
        culture ??= CultureInfo.CurrentCulture;
        if (num < -10000000) { num /= 10000; return (num / 100m).ToString("#.##'M'", culture); }
        if (num < -1000000) { num /= 100; return (num / 10m).ToString("#.##'K'", culture); }
        if (num < -10000) { num /= 10; return (num / 100m).ToString("#.##'K'", culture); }
        if (num < 1000) return num.ToString("N0", culture);
        if (num < 10000) { num /= 10; return (num / 100m).ToString("#.##'K'", culture); }
        if (num < 1000000) { num /= 100; return (num / 10m).ToString("#.##'K'", culture); }
        if (num < 10000000) { num /= 10000; return (num / 100m).ToString("#.##'M'", culture); }
        num /= 100000;
        return (num / 10m).ToString("#.##'M'", culture);
    }

    // Player Objects Aggregation
    public static long GetCurrentTotalDamage(this List<KeyValuePair<Guid, PlayerGameObject>> playerObjects)
        => playerObjects.Count <= 0 ? 0 : playerObjects.Max(x => x.Value.Damage);

    public static long GetCurrentTotalHeal(this List<KeyValuePair<Guid, PlayerGameObject>> playerObjects)
        => playerObjects.Count <= 0 ? 0 : playerObjects.Max(x => x.Value.Heal);

    public static long GetCurrentTotalTakenDamage(this List<KeyValuePair<Guid, PlayerGameObject>> playerObjects)
        => playerObjects.Count <= 0 ? 0 : playerObjects.Max(x => x.Value.TakenDamage);

    public static double GetDamagePercentage(this List<KeyValuePair<Guid, PlayerGameObject>> playerObjects, double playerDamage)
    {
        var totalDamage = playerObjects.Sum(x => x.Value.Damage);
        return totalDamage == 0 ? 0 : 100.00 / totalDamage * playerDamage;
    }

    public static double GetHealPercentage(this List<KeyValuePair<Guid, PlayerGameObject>> playerObjects, double playerHeal)
    {
        var totalHeal = playerObjects.Sum(x => x.Value.Heal);
        return totalHeal == 0 ? 0 : 100.00 / totalHeal * playerHeal;
    }

    public static double GetTakenDamagePercentage(this List<KeyValuePair<Guid, PlayerGameObject>> playerObjects, double playerDamage)
    {
        var totalTakenDamage = playerObjects.Sum(x => x.Value.TakenDamage);
        return totalTakenDamage == 0 ? 0 : 100.00 / totalTakenDamage * playerDamage;
    }
}
