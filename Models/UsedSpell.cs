using AlbionInfoTracker.Enums;

namespace AlbionInfoTracker.Models;

public class UsedSpell
{
    public UsedSpell(int spellIndex, int itemIndex)
    {
        SpellIndex = spellIndex;
        ItemIndex = itemIndex;
    }

    public int SpellIndex { get; init; }
    public int ItemIndex { get; init; }
    public string UniqueName { get; init; } = string.Empty;
    public string Target { get; init; } = string.Empty;
    public string Category { get; init; } = string.Empty;

    public HealthChangeType HealthChangeType { get; set; }
    public long DamageHealValue { get; set; }
    public int Ticks { get; set; }
}
