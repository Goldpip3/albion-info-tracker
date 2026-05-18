using AlbionInfoTracker.Enums;
using AlbionInfoTracker.Models;
using Serilog;
using System;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Linq;

namespace AlbionInfoTracker.Services;

public class EntityController
{
    private readonly ConcurrentDictionary<Guid, PlayerGameObject> _knownEntities = new();
    private readonly ConcurrentDictionary<long, CharacterEquipment> _pendingEquipment = new();

    public event Action? OnProfileOrPartyChanged;

    #region Entities

    public void AddEntity(Entity entity)
    {
        PlayerGameObject gameObject;

        if (_knownEntities.TryRemove(entity.UserGuid, out var oldEntity))
        {
            long? newUserObjectId = oldEntity.ObjectId;
            if (entity.ObjectId != null)
                newUserObjectId = entity.ObjectId;

            gameObject = new PlayerGameObject(newUserObjectId)
            {
                Name = entity.Name,
                ObjectType = entity.ObjectType,
                UserGuid = entity.UserGuid,
                Guild = string.Empty == entity.Guild ? oldEntity.Guild : entity.Guild,
                Alliance = string.Empty == entity.Alliance ? oldEntity.Alliance : entity.Alliance,
                InteractGuid = entity.InteractGuid == Guid.Empty || entity.InteractGuid == null ? oldEntity.InteractGuid : entity.InteractGuid,
                ObjectSubType = entity.ObjectSubType,
                CharacterEquipment = entity.CharacterEquipment ?? oldEntity.CharacterEquipment,
                CombatStart = oldEntity.CombatStart,
                CombatTime = oldEntity.CombatTime,
                Damage = oldEntity.Damage,
                Heal = oldEntity.Heal,
                Overhealed = oldEntity.Overhealed,
                IsInParty = oldEntity.IsInParty,
                Spells = oldEntity.Spells
            };
        }
        else
        {
            gameObject = new PlayerGameObject(entity.ObjectId)
            {
                Name = entity.Name,
                ObjectType = entity.ObjectType,
                UserGuid = entity.UserGuid,
                Guild = entity.Guild,
                Alliance = entity.Alliance,
                ObjectSubType = entity.ObjectSubType,
                CharacterEquipment = entity.CharacterEquipment
            };

            if (gameObject.ObjectSubType == GameObjectSubType.LocalPlayer)
                RemoveLocalEntityFromParty();
        }

        if (gameObject.Name == "PA" && oldEntity?.Name != null)
            gameObject.Name = oldEntity.Name;

        // Apply any pending equipment that arrived before the entity was created
        if (gameObject.ObjectId != null && _pendingEquipment.TryRemove((long)gameObject.ObjectId, out var pendingEquip))
        {
            gameObject.CharacterEquipment = pendingEquip;
            Log.Debug("AddEntity: applied pending equipment for objectId={ObjectId} mainHand={MainHand}",
                gameObject.ObjectId, pendingEquip.MainHand);
        }

        _knownEntities.TryAdd(gameObject.UserGuid, gameObject);

        Log.Debug("AddEntity: name={Name} objectId={ObjectId} subType={SubType} mainHand={MainHand}",
            gameObject.Name, gameObject.ObjectId, gameObject.ObjectSubType, gameObject.CharacterEquipment?.MainHand);

        if (gameObject.ObjectSubType == GameObjectSubType.LocalPlayer)
            OnProfileOrPartyChanged?.Invoke();
    }

    public KeyValuePair<Guid, PlayerGameObject>? GetEntity(long objectId)
        => _knownEntities.FirstOrDefault(x => x.Value.ObjectId == objectId);

    public KeyValuePair<Guid, PlayerGameObject> GetEntity(Guid guid)
        => _knownEntities.FirstOrDefault(x => x.Key == guid);

    public void SetCharacterEquipment(long objectId, CharacterEquipment equipment)
    {
        var entity = GetEntity(objectId);
        if (entity?.Value != null)
        {
            entity.Value.Value.CharacterEquipment = equipment;
            Log.Debug("SetCharacterEquipment: objectId={ObjectId} name={Name} mainHand={MainHand}",
                objectId, entity.Value.Value.Name, equipment.MainHand);
        }
        else
        {
            // Entity not yet created — cache equipment for when it arrives
            _pendingEquipment[objectId] = equipment;
            Log.Debug("SetCharacterEquipment: cached pending equipment for objectId={ObjectId} mainHand={MainHand}",
                objectId, equipment.MainHand);
        }
    }

    public List<KeyValuePair<Guid, PlayerGameObject>> GetAllEntitiesWithDamageOrHealAndInParty()
    {
        return new List<KeyValuePair<Guid, PlayerGameObject>>(_knownEntities
            .ToArray()
            .Where(x => (x.Value.Damage > 0 || x.Value.Heal > 0 || x.Value.Overhealed > 0) && IsEntityInParty(x.Key)));
    }

    public bool ExistEntity(Guid guid)
        => _knownEntities.Any(x => x.Key == guid);

    #endregion

    #region Party

    public void AddToParty(Guid guid)
    {
        var entity = GetEntity(guid);
        if (entity.Value is { IsInParty: false })
        {
            entity.Value.IsInParty = true;
            OnProfileOrPartyChanged?.Invoke();
        }
    }

    private void RemoveLocalEntityFromParty()
    {
        var entity = GetLocalEntity();
        if (entity?.Value != null)
            RemoveFromParty(entity.Value.Key);
    }

    public void RemoveFromParty(Guid? guid)
    {
        if (guid is not { } notNullGuid) return;

        if (notNullGuid == GetLocalEntity()?.Key)
        {
            ResetPartyMember();
            AddLocalEntityToParty();
        }
        else
        {
            var entity = GetEntity(notNullGuid);
            if (entity.Value != null)
                entity.Value.IsInParty = false;
        }
        OnProfileOrPartyChanged?.Invoke();
    }

    public void ResetPartyMember()
    {
        foreach (var partyEntities in _knownEntities.Where(x => x.Value.IsInParty))
            partyEntities.Value.IsInParty = false;
    }

    public void AddLocalEntityToParty()
    {
        var localEntity = GetLocalEntity();
        if (localEntity?.Value != null)
            localEntity.Value.Value.IsInParty = true;
    }

    public void SetParty(Dictionary<Guid, string> party)
    {
        ResetPartyMember();
        foreach (var member in party)
        {
            if (!ExistEntity(member.Key) && GetLocalEntity()?.Key != member.Key)
            {
                AddEntity(new Entity
                {
                    UserGuid = member.Key,
                    Name = member.Value,
                    ObjectType = GameObjectType.Player,
                    ObjectSubType = GameObjectSubType.Player
                });
            }
            AddToParty(member.Key);
        }
        OnProfileOrPartyChanged?.Invoke();
    }

    public bool IsEntityInParty(string? name)
    {
        if (string.IsNullOrEmpty(name)) return false;
        return _knownEntities.FirstOrDefault(x => x.Value?.Name == name).Value?.IsInParty ?? false;
    }

    public bool IsEntityInParty(long objectId)
    {
        var entity = _knownEntities.FirstOrDefault(x => x.Value.ObjectId == objectId);
        return IsEntityInParty(entity.Value?.Name);
    }

    public bool IsEntityInParty(Guid guid)
        => _knownEntities.FirstOrDefault(x => x.Key == guid).Value?.IsInParty ?? false;

    public List<string> GetPartyMemberNames()
    {
        return _knownEntities.Where(x => x.Value.IsInParty).Select(x => x.Value.Name).ToList();
    }

    public int GetPartyMemberCount()
    {
        return _knownEntities.Count(x => x.Value.IsInParty);
    }

    public List<KeyValuePair<Guid, PlayerGameObject>> GetAllEntitiesInParty()
    {
        return _knownEntities.ToArray().Where(x => x.Value.IsInParty).ToList();
    }

    #endregion

    #region Damage Reset

    public void ResetEntitiesDamageStartTime()
    {
        foreach (var entity in _knownEntities) entity.Value.CombatStart = null;
    }

    public void ResetEntitiesDamageTimes()
    {
        foreach (var entity in _knownEntities) entity.Value.ResetCombatTimes();
    }

    public void ResetEntitiesDamage()
    {
        foreach (var entity in _knownEntities) entity.Value.Damage = 0;
    }

    public void ResetEntitiesHeal()
    {
        foreach (var entity in _knownEntities) entity.Value.Heal = 0;
    }

    public void ResetEntitiesTakeDamage()
    {
        foreach (var entity in _knownEntities) entity.Value.TakenDamage = 0;
    }

    public void ResetSpells()
    {
        foreach (var entity in _knownEntities) entity.Value.Spells = new List<UsedSpell>();
    }

    public void ResetEntitiesHealAndOverhealed()
    {
        foreach (var entity in _knownEntities) entity.Value.Overhealed = 0;
    }

    #endregion

    #region Local Entity

    public bool ExistLocalEntity()
        => _knownEntities.Any(x => x.Value.ObjectSubType == GameObjectSubType.LocalPlayer);

    public KeyValuePair<Guid, PlayerGameObject>? GetLocalEntity()
        => _knownEntities.ToArray().FirstOrDefault(x => x.Value.ObjectSubType == GameObjectSubType.LocalPlayer);

    #endregion
}
