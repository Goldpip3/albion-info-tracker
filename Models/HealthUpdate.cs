using AlbionInfoTracker.Enums;

namespace AlbionInfoTracker.Models;

public class HealthUpdate
{
    public long AffectedObjectId { get; set; }
    public double HealthChange { get; set; }
    public double NewHealthValue { get; set; }
    public EffectType EffectType { get; set; }
    public EffectOrigin EffectOrigin { get; set; }
    public long CauserId { get; set; }
    public int CausingSpellIndex { get; set; }
}
