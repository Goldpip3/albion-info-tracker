using AlbionInfoTracker.Enums;
using AlbionInfoTracker.Models;
using Serilog;
using System;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;

namespace AlbionInfoTracker.Services;

public class CombatController
{
    private readonly EntityController _entityController;
    private static bool _isUiUpdateActive;
    private DateTime _lastDamageUiUpdate;

    public ConcurrentDictionary<Guid, double> LastPlayersHealth = new();

    public event Action<List<KeyValuePair<Guid, PlayerGameObject>>>? OnDamageUpdate;
    public event Action<long, bool, bool>? OnChangeCombatMode;
    public event Action<long, long, long>? OnFameOrSilverUpdate;

    private long _sessionFame;
    private long _sessionCombatFame;
    private long _sessionSilver;

    public long SessionFame => _sessionFame;
    public long SessionCombatFame => _sessionCombatFame;
    public long SessionSilver => _sessionSilver;

    public bool IsTrackingActive { get; set; } = true;
    public bool IsResetByMapChangeActive { get; set; }

    public CombatController(EntityController entityController)
    {
        _entityController = entityController;
        OnChangeCombatMode += AddCombatTime;
    }

    #region Damage Meter methods

    public Task AddDamage(long affectedId, long causerId, double healthChange, double newHealthValue, int causingSpellIndex)
    {
        var healthChangeType = GetHealthChangeType(healthChange);
        if (!IsTrackingActive || (affectedId == causerId && healthChangeType == HealthChangeType.Damage))
            return Task.CompletedTask;

        var causerGameObject = _entityController.GetEntity(causerId);
        var causerGameObjectValue = causerGameObject?.Value;

        if (causerGameObject?.Value is not { ObjectType: GameObjectType.Player } || !_entityController.IsEntityInParty(causerGameObject.Value.Key))
            return Task.CompletedTask;

        if (healthChangeType == HealthChangeType.Damage)
        {
            var damageChangeValue = (int)Math.Round(healthChange.ToPositiveFromNegativeOrZero(), MidpointRounding.AwayFromZero);
            if (damageChangeValue <= 0) return Task.CompletedTask;

            causerGameObjectValue!.Damage += damageChangeValue;
            AddOrUpdateSpell(causingSpellIndex, causerGameObjectValue, healthChangeType, damageChangeValue, causerGameObjectValue.CharacterEquipment?.MainHand ?? 0);
        }

        if (healthChangeType == HealthChangeType.Heal)
        {
            var healChangeValue = healthChange;
            if (healChangeValue <= 0) return Task.CompletedTask;

            var positiveHealChangeValue = (int)Math.Round(healChangeValue, MidpointRounding.AwayFromZero);
            if (!IsMaxHealthReached(affectedId, newHealthValue))
            {
                causerGameObjectValue!.Heal += positiveHealChangeValue;
                AddOrUpdateSpell(causingSpellIndex, causerGameObjectValue, healthChangeType, positiveHealChangeValue, causerGameObjectValue.CharacterEquipment?.MainHand ?? 0);
            }
            else
            {
                causerGameObjectValue!.Overhealed += positiveHealChangeValue;
            }
        }

        causerGameObjectValue!.CombatStart ??= DateTime.UtcNow;

        if (IsUiUpdateAllowed())
        {
            var entities = _entityController.GetAllEntitiesWithDamageOrHealAndInParty();
            OnDamageUpdate?.Invoke(entities);
        }

        return Task.CompletedTask;
    }

    public Task AddTakenDamage(long affectedId, long causerId, double healthChange, double newHealthValue, int causingSpellIndex)
    {
        var healthChangeType = GetHealthChangeType(healthChange);
        if (!IsTrackingActive || (affectedId == causerId && healthChangeType == HealthChangeType.Damage))
            return Task.CompletedTask;

        var gameObject = _entityController.GetEntity(affectedId);
        var gameObjectValue = gameObject?.Value;

        if (gameObject?.Value is not { ObjectType: GameObjectType.Player } || !_entityController.IsEntityInParty(gameObject.Value.Key))
            return Task.CompletedTask;

        if (healthChangeType == HealthChangeType.Damage)
        {
            var damageChangeValue = (int)Math.Round(healthChange.ToPositiveFromNegativeOrZero(), MidpointRounding.AwayFromZero);
            if (damageChangeValue <= 0) return Task.CompletedTask;
            gameObjectValue!.TakenDamage += damageChangeValue;
        }

        return Task.CompletedTask;
    }

    public Task AddFame(long totalFame)
    {
        if (!IsTrackingActive) return Task.CompletedTask;
        _sessionFame += totalFame;
        OnFameOrSilverUpdate?.Invoke(_sessionFame, _sessionCombatFame, _sessionSilver);
        return Task.CompletedTask;
    }

    public Task AddCombatFame(long combatFame)
    {
        if (!IsTrackingActive) return Task.CompletedTask;
        _sessionCombatFame += combatFame;
        OnFameOrSilverUpdate?.Invoke(_sessionFame, _sessionCombatFame, _sessionSilver);
        return Task.CompletedTask;
    }

    public Task AddSilver(long silver)
    {
        if (!IsTrackingActive) return Task.CompletedTask;
        _sessionSilver += silver;
        OnFameOrSilverUpdate?.Invoke(_sessionFame, _sessionCombatFame, _sessionSilver);
        return Task.CompletedTask;
    }

    public void UpdateCombatMode(long objectId, bool inActiveCombat, bool inPassiveCombat)
    {
        OnChangeCombatMode?.Invoke(objectId, inActiveCombat, inPassiveCombat);
    }

    public void ResetDamageMeterByClusterChange()
    {
        if (!IsResetByMapChangeActive) return;
        ResetDamageMeter();
        LastPlayersHealth.Clear();
    }

    public void ResetDamageMeter()
    {
        _entityController.ResetEntitiesDamageTimes();
        _entityController.ResetEntitiesDamage();
        _entityController.ResetEntitiesHeal();
        _entityController.ResetEntitiesTakeDamage();
        _entityController.ResetSpells();
        _entityController.ResetEntitiesHealAndOverhealed();
        _entityController.ResetEntitiesDamageStartTime();

        _sessionFame = 0;
        _sessionCombatFame = 0;
        _sessionSilver = 0;
        OnFameOrSilverUpdate?.Invoke(0, 0, 0);
    }

    public bool IsMaxHealthReached(long objectId, double newHealthValue)
    {
        var gameObject = _entityController.GetEntity(objectId);
        var playerHealth = LastPlayersHealth.ToArray().FirstOrDefault(x => x.Key == gameObject?.Value?.UserGuid);
        if (playerHealth.Value.CompareTo(newHealthValue) == 0) return true;
        SetLastPlayersHealth(gameObject?.Value?.UserGuid, newHealthValue);
        return false;
    }

    private void SetLastPlayersHealth(Guid? userGuid, double value)
    {
        if (userGuid is not { } notNullGuid) return;
        LastPlayersHealth.AddOrUpdate(notNullGuid, value, (_, _) => value);
    }

    private static HealthChangeType GetHealthChangeType(double healthChange) => healthChange <= 0 ? HealthChangeType.Damage : HealthChangeType.Heal;

    private bool IsUiUpdateAllowed(int waitTimeInSeconds = 1)
    {
        var currentDateTime = DateTime.UtcNow;
        var difference = currentDateTime.Subtract(_lastDamageUiUpdate);
        if (difference.Seconds >= waitTimeInSeconds && !_isUiUpdateActive)
        {
            _lastDamageUiUpdate = currentDateTime;
            return true;
        }
        return false;
    }

    private void AddOrUpdateSpell(int causingSpellIndex, PlayerGameObject playerGameObject, HealthChangeType healthChangeType, int healthChangeValue, int itemIndex)
    {
        if (causingSpellIndex <= 0)
        {
            var autoAttack = playerGameObject.Spells.FirstOrDefault(x => x.SpellIndex == 0);
            if (autoAttack is not null)
            {
                autoAttack.DamageHealValue += healthChangeValue;
                autoAttack.Ticks++;
            }
            else
            {
                playerGameObject.Spells.Add(new UsedSpell(0, 0)
                {
                    UniqueName = "AUTO_ATTACK",
                    Category = "damage",
                    DamageHealValue = healthChangeValue,
                    HealthChangeType = healthChangeType,
                    Ticks = 1
                });
            }
            return;
        }

        var spell = playerGameObject.Spells.FirstOrDefault(x => x.SpellIndex == causingSpellIndex && x.HealthChangeType == healthChangeType);
        if (spell is not null)
        {
            spell.HealthChangeType = healthChangeType;
            spell.DamageHealValue += healthChangeValue;
            spell.Ticks++;
        }
        else
        {
            playerGameObject.Spells.Add(new UsedSpell(causingSpellIndex, itemIndex)
            {
                HealthChangeType = healthChangeType,
                DamageHealValue = healthChangeValue,
                Ticks = 1
            });
        }
    }

    public static double GetOverhealedPercentageOfHealWithOverhealed(double overhealed, double heal)
    {
        var total = heal + overhealed;
        return total == 0 ? 0 : 100.00 / total * overhealed;
    }

    #endregion

    #region Combat Timer

    private void AddCombatTime(long objectId, bool inActiveCombat, bool inPassiveCombat)
    {
        if (!_entityController.IsEntityInParty(objectId)) return;

        var playerObject = _entityController.GetEntity(objectId);
        if (playerObject?.Value == null) return;

        if ((inActiveCombat || inPassiveCombat) && playerObject.Value.Value.CombatTimes.Any(x => x?.EndTime == null))
            return;

        if (inActiveCombat || inPassiveCombat)
            playerObject.Value.Value.AddCombatTime(new ActionInterval(DateTime.UtcNow));

        if (!inActiveCombat && !inPassiveCombat)
        {
            var combatTime = playerObject.Value.Value.CombatTimes.FirstOrDefault(x => x.EndTime == null);
            if (combatTime != null)
                combatTime.EndTime = DateTime.UtcNow;
        }
    }

    #endregion
}
