using StatisticsAnalysisTool.Common;
using StatisticsAnalysisTool.Localization;
using StatisticsAnalysisTool.Models;
using StatisticsAnalysisTool.ViewModels;
using System;
using System.Collections.ObjectModel;
using System.Windows;
using System.Windows.Input;

namespace StatisticsAnalysisTool.DamageMeter;

public class DamageMeterFragment : BaseViewModel
{
    private string _shopSubCategory;
    private Guid _causerGuid;
    private Item _causerMainHand;
    private long _damage;
    private double _damageInPercent;
    private double _damagePercentage;
    private double _dps;
    private string _dpsString;
    private string _name;
    private long _heal;
    private string _hpsString;
    private double _hps;
    private double _healInPercent;
    private double _healPercentage;
    private bool _isDamageMeterShowing = true;
    private string _damageShortString;
    private string _healShortString;
    private TimeSpan _combatTime;
    private double _overhealedPercentageOfTotalHealing;
    private double _overhealed;
    private long _takenDamage;
    private string _takenDamageShortString;
    private double _takenDamageInPercent;
    private double _takenDamagePercentage;
    private long _currentDamage;
    private string _currentDamageShortString = "0";
    private double _currentDps;
    private string _currentDpsString = "0";
    private long _currentHeal;
    private string _currentHealShortString = "0";
    private double _currentHps;
    private string _currentHpsString = "0";
    private long _currentTakenDamage;
    private string _currentTakenDamageShortString = "0";
    private DamageMeterStyleFragmentType _damageMeterStyleFragmentType;
    private Visibility _spellsContainerVisibility = Visibility.Collapsed;
    private ObservableCollection<UsedSpellFragment> _spells = new();

    public DamageMeterFragment(DamageMeterFragment damageMeterFragment)
    {
        CauserGuid = damageMeterFragment.CauserGuid;
        Damage = damageMeterFragment.Damage;
        Dps = damageMeterFragment.Dps;
        DamageInPercent = damageMeterFragment.DamageInPercent;
        DamagePercentage = damageMeterFragment.DamagePercentage;
        Heal = damageMeterFragment.Heal;
        Hps = damageMeterFragment.Hps;
        HealInPercent = damageMeterFragment.HealInPercent;
        HealPercentage = damageMeterFragment.HealPercentage;
        Name = damageMeterFragment.Name;
        CauserMainHand = damageMeterFragment.CauserMainHand;
        Spells = damageMeterFragment.Spells;
        TakenDamage = damageMeterFragment.TakenDamage;
        TakenDamageInPercent = damageMeterFragment.TakenDamageInPercent;
        TakenDamagePercentage = damageMeterFragment.TakenDamagePercentage;
    }

    public DamageMeterFragment()
    {
    }

    public string Name
    {
        get => _name;
        set
        {
            _name = value;
            OnPropertyChanged();
        }
    }

    public Guid CauserGuid
    {
        get => _causerGuid;
        set
        {
            _causerGuid = value;
            OnPropertyChanged();
        }
    }

    public bool IsDamageMeterShowing
    {
        get => _isDamageMeterShowing;
        set
        {
            _isDamageMeterShowing = value;
            OnPropertyChanged();
        }
    }

    public DamageMeterStyleFragmentType DamageMeterStyleFragmentType
    {
        get => _damageMeterStyleFragmentType;
        set
        {
            _damageMeterStyleFragmentType = value;
            OnPropertyChanged();
        }
    }

    public TimeSpan CombatTime
    {
        get => _combatTime;
        set
        {
            _combatTime = value;
            OnPropertyChanged();
        }
    }

    #region Damage

    public long Damage
    {
        get => _damage;
        set
        {
            _damage = value;
            DamageShortString = _damage.ToShortNumberString();
            OnPropertyChanged();
        }
    }

    public string DamageShortString
    {
        get => _damageShortString;
        private set
        {
            _damageShortString = value;
            OnPropertyChanged();
            OnPropertyChanged(nameof(DamageDualString));
        }
    }

    public string DpsString
    {
        get => _dpsString;
        private set
        {
            _dpsString = value;
            OnPropertyChanged();
            OnPropertyChanged(nameof(DpsDualString));
        }
    }

    public double Dps
    {
        get => _dps;
        set
        {
            _dps = value;
            DpsString = _dps.ToShortNumberString();
            OnPropertyChanged();
        }
    }

    public double DamageInPercent
    {
        get => _damageInPercent;
        set
        {
            _damageInPercent = value;
            OnPropertyChanged();
        }
    }

    public double DamagePercentage
    {
        get => _damagePercentage;
        set
        {
            _damagePercentage = value;
            OnPropertyChanged();
        }
    }

    #endregion

    #region Heal

    public long Heal
    {
        get => _heal;
        set
        {
            _heal = value;
            HealShortString = _heal.ToShortNumberString();
            OnPropertyChanged();
        }
    }

    public string HealShortString
    {
        get => _healShortString;
        private set
        {
            _healShortString = value;
            OnPropertyChanged();
            OnPropertyChanged(nameof(HealDualString));
        }
    }

    public string HpsString
    {
        get => _hpsString;
        private set
        {
            _hpsString = value;
            OnPropertyChanged();
            OnPropertyChanged(nameof(HpsDualString));
        }
    }

    public double Hps
    {
        get => _hps;
        set
        {
            _hps = value;
            HpsString = _hps.ToShortNumberString();
            OnPropertyChanged();
        }
    }

    public double HealInPercent
    {
        get => _healInPercent;
        set
        {
            _healInPercent = value;
            OnPropertyChanged();
        }
    }

    public double HealPercentage
    {
        get => _healPercentage;
        set
        {
            _healPercentage = value;
            OnPropertyChanged();
        }
    }

    public double Overhealed
    {
        get => _overhealed;
        set
        {
            _overhealed = value;
            OnPropertyChanged();
        }
    }

    public double OverhealedPercentageOfTotalHealing
    {
        get => _overhealedPercentageOfTotalHealing;
        set
        {
            _overhealedPercentageOfTotalHealing = value;
            OnPropertyChanged();
        }
    }

    #endregion

    #region Take Damage

    public long TakenDamage
    {
        get => _takenDamage;
        set
        {
            _takenDamage = value;
            TakenDamageShortString = _takenDamage.ToShortNumberString();
            OnPropertyChanged();
        }
    }

    public string TakenDamageShortString
    {
        get => _takenDamageShortString;
        private set
        {
            _takenDamageShortString = value;
            OnPropertyChanged();
            OnPropertyChanged(nameof(TakenDamageDualString));
        }
    }

    public double TakenDamageInPercent
    {
        get => _takenDamageInPercent;
        set
        {
            _takenDamageInPercent = value;
            OnPropertyChanged();
        }
    }

    public double TakenDamagePercentage
    {
        get => _takenDamagePercentage;
        set
        {
            _takenDamagePercentage = value;
            OnPropertyChanged();
        }
    }

    #endregion

    #region Current fight (Skada-style: live event, or last completed if out of combat)

    public long CurrentDamage
    {
        get => _currentDamage;
        set
        {
            _currentDamage = value;
            CurrentDamageShortString = _currentDamage.ToShortNumberString();
            OnPropertyChanged();
        }
    }

    public string CurrentDamageShortString
    {
        get => _currentDamageShortString;
        private set
        {
            _currentDamageShortString = value;
            OnPropertyChanged();
            OnPropertyChanged(nameof(DamageDualString));
        }
    }

    public double CurrentDps
    {
        get => _currentDps;
        set
        {
            _currentDps = value;
            CurrentDpsString = _currentDps.ToShortNumberString();
            OnPropertyChanged();
        }
    }

    public string CurrentDpsString
    {
        get => _currentDpsString;
        private set
        {
            _currentDpsString = value;
            OnPropertyChanged();
            OnPropertyChanged(nameof(DpsDualString));
        }
    }

    public long CurrentHeal
    {
        get => _currentHeal;
        set
        {
            _currentHeal = value;
            CurrentHealShortString = _currentHeal.ToShortNumberString();
            OnPropertyChanged();
        }
    }

    public string CurrentHealShortString
    {
        get => _currentHealShortString;
        private set
        {
            _currentHealShortString = value;
            OnPropertyChanged();
            OnPropertyChanged(nameof(HealDualString));
        }
    }

    public double CurrentHps
    {
        get => _currentHps;
        set
        {
            _currentHps = value;
            CurrentHpsString = _currentHps.ToShortNumberString();
            OnPropertyChanged();
        }
    }

    public string CurrentHpsString
    {
        get => _currentHpsString;
        private set
        {
            _currentHpsString = value;
            OnPropertyChanged();
            OnPropertyChanged(nameof(HpsDualString));
        }
    }

    public long CurrentTakenDamage
    {
        get => _currentTakenDamage;
        set
        {
            _currentTakenDamage = value;
            CurrentTakenDamageShortString = _currentTakenDamage.ToShortNumberString();
            OnPropertyChanged();
        }
    }

    public string CurrentTakenDamageShortString
    {
        get => _currentTakenDamageShortString;
        private set
        {
            _currentTakenDamageShortString = value;
            OnPropertyChanged();
            OnPropertyChanged(nameof(TakenDamageDualString));
        }
    }

    public string DamageDualString => $"{_currentDamageShortString ?? "0"} | {_damageShortString ?? "0"}";
    public string DpsDualString => $"{_currentDpsString ?? "0"} | {_dpsString ?? "0"} dps";
    public string HealDualString => $"{_currentHealShortString ?? "0"} | {_healShortString ?? "0"}";
    public string HpsDualString => $"{_currentHpsString ?? "0"} | {_hpsString ?? "0"} hps";
    public string TakenDamageDualString => $"{_currentTakenDamageShortString ?? "0"} | {_takenDamageShortString ?? "0"}";

    #endregion

    #region Spells

    public ObservableCollection<UsedSpellFragment> Spells
    {
        get => _spells;
        set
        {
            _spells = value;
            OnPropertyChanged();
        }
    }

    public Visibility SpellsContainerVisibility
    {
        get => _spellsContainerVisibility;
        set
        {
            _spellsContainerVisibility = value;
            OnPropertyChanged();
        }
    }

    #endregion

    public Item CauserMainHand
    {
        get => _causerMainHand;
        set
        {
            _causerMainHand = value;
            ShopSubCategory = _causerMainHand?.FullItemInformation?.ShopSubCategory1;
            OnPropertyChanged();
        }
    }

    public string ShopSubCategory
    {
        get => _shopSubCategory;
        set
        {
            _shopSubCategory = value;
            OnPropertyChanged();
        }
    }

    private void PerformShowSpells(object value)
    {
        SpellsContainerVisibility = SpellsContainerVisibility == Visibility.Visible ? Visibility.Collapsed : Visibility.Visible;
    }

    private ICommand _showSpells;
    public ICommand ShowSpells => _showSpells ??= new CommandHandler(PerformShowSpells, true);

    public static string TranslationCombatTime => LocalizationController.Translation("COMBAT_TIME");
    public static string TranslationHealingWithoutOverhealed => LocalizationController.Translation("HEALING_WITHOUT_OVERHEALED");
    public static string TranslationOverhealedPercentageOfTotalHealing => LocalizationController.Translation("OVERHEALED_PERCENTAGE_OF_TOTAL_HEALING");
    public static string TranslationDmgPercent => LocalizationController.Translation("DMG_PERCENT");
    public static string TranslationName => LocalizationController.Translation("NAME");
    public static string TranslationDamageHeal => LocalizationController.Translation("DAMAGE_HEAL");
    public static string TranslationTicks => LocalizationController.Translation("TICKS");

    public override bool Equals(object obj)
    {
        return obj is DamageMeterFragment damageMeterFragment && Name == damageMeterFragment.Name
                                                              && Damage == damageMeterFragment.Damage
                                                              && CauserGuid == damageMeterFragment.CauserGuid
                                                              && Heal == damageMeterFragment.Heal;
    }

    public override int GetHashCode()
    {
        return HashCode.Combine(Name, CauserGuid, Damage, Heal);
    }
}