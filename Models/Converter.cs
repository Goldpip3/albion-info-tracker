using System;
using System.Collections.Generic;

namespace AlbionInfoTracker.Models;

public static class Converter
{
    public static Dictionary<int, TOut> GetValue<TOut>(object parameter) where TOut : struct
    {
        var dictionary = new Dictionary<int, TOut>();
        if (parameter == null) return dictionary;

        var valueType = parameter.GetType();
        if (!valueType.IsArray) return dictionary;

        var array = (Array)parameter;
        for (int i = 0; i < array.Length; i++)
        {
            var value = array.GetValue(i);
            var convertedValue = ConvertTo<TOut>(value);
            if (convertedValue.HasValue)
            {
                dictionary.Add(i, convertedValue.Value);
            }
        }
        return dictionary;
    }

    private static TOut? ConvertTo<TOut>(object? input) where TOut : struct
    {
        if (input == null) return null;
        return typeof(TOut).Name switch
        {
            nameof(Byte) => (TOut)(object)input.ObjectToByte(),
            nameof(Int16) => (TOut)(object)input.ObjectToShort(),
            nameof(Int32) => (TOut)(object)input.ObjectToInt(),
            nameof(Int64) => (TOut)(object)(input.ObjectToLong() ?? 0L),
            nameof(UInt64) => (TOut)(object)(input.ObjectToUlong() ?? 0UL),
            nameof(Double) => (TOut)(object)input.ObjectToDouble(),
            _ => null
        };
    }
}
