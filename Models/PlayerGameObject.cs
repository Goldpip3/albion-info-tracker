using System;
using System.Collections.Generic;
using System.Linq;

namespace AlbionInfoTracker.Models;

public class PlayerGameObject : GameObject
{
    private CharacterEquipment? _characterEquipment;
    private readonly Guid _userGuid;
    private Guid? _interactGuid;
    private List<ActionInterval> _combatTimes = new();

    public PlayerGameObject(long? objectId)
    {
        ObjectId ??= objectId;
        LastUpdate = DateTime.UtcNow.Ticks;
    }

    public long LastUpdate { get; private set; }

    public Guid UserGuid
    {
        get => _userGuid;
        init
        {
            _userGuid = value;
            LastUpdate = DateTime.UtcNow.Ticks;
        }
    }

    public Guid? InteractGuid
    {
        get => _interactGuid;
        set
        {
            _interactGuid = value;
            LastUpdate = DateTime.UtcNow.Ticks;
        }
    }

    public string Name { get; set; } = "Unknown";
    public string Guild { get; set; } = string.Empty;
    public string Alliance { get; set; } = string.Empty;
    public bool IsInParty { get; set; }
    public double ItemPower { get; set; }

    public CharacterEquipment? CharacterEquipment
    {
        get => _characterEquipment;
        set
        {
            _characterEquipment = value;
            LastUpdate = DateTime.UtcNow.Ticks;
        }
    }

    public DateTime? CombatStart { get; set; }

    public List<ActionInterval> CombatTimes
    {
        get => _combatTimes;
        set
        {
            _combatTimes = value;
            LastUpdate = DateTime.UtcNow.Ticks;
        }
    }

    public TimeSpan CombatTime { get; set; } = new(1);
    public long Damage { get; set; }
    public long Heal { get; set; }
    public long TakenDamage { get; set; }
    public List<UsedSpell> Spells { get; set; } = new();
    public long Overhealed { get; set; }

    public double Dps => GetValuePerSecond(Damage);
    public double Hps => GetValuePerSecond(Heal);

    private double GetValuePerSecond(double value, double maxValue = 9999)
    {
        if (double.IsInfinity(value)) return maxValue;

        if (CombatTime.Ticks <= 1 && CombatStart != null)
        {
            var startTimeSpan = DateTime.UtcNow - (DateTime)CombatStart;
            var calculation = value / startTimeSpan.TotalSeconds;
            return calculation > maxValue ? maxValue : calculation;
        }

        var valuePerSeconds = value / CombatTime.TotalSeconds;
        if (maxValue > 0 && valuePerSeconds > maxValue) return maxValue;
        return valuePerSeconds;
    }

    public override string ToString() => $"{ObjectType}[ObjectId: {ObjectId}, Name: '{Name}']";

    #region Combat

    public void AddCombatTime(ActionInterval actionInterval)
    {
        CombatTimes.Add(actionInterval);
        SetCombatTimeSpan();
    }

    public void ResetCombatTimes()
    {
        CombatTimes.Clear();
        CombatTime = new TimeSpan();
    }

    private void SetCombatTimeSpan()
    {
        foreach (var combatTime in CombatTimes.Where(x => x.EndTime != null).ToList())
        {
            CombatTime += combatTime.TimeSpan;
            CombatTimes.Remove(combatTime);
        }
    }

    #endregion
}
