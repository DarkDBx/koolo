package action

import (
	"slices"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/item"
	"github.com/hectorgimenez/d2go/pkg/data/stat"
	"github.com/hectorgimenez/koolo/internal/action/step"
	"github.com/hectorgimenez/koolo/internal/context"
	"github.com/hectorgimenez/koolo/internal/game"
	"github.com/hectorgimenez/koolo/internal/ui"
	"github.com/hectorgimenez/koolo/internal/utils"
)

type SocketRecipe struct {
	Name          string
	Inserts       []string
	BaseItemTypes []string
	BaseSockets   int
}

var (
	SocketRecipes = []SocketRecipe{
		// Add recipes in order of priority. If we have inserts for two recipes, it will process the first one in the list first.
		// Add the inserts in order required for the runeword
		{
			Name:          "TirTir",
			Inserts:       []string{"TirRune", "TirRune"},
			BaseItemTypes: []string{"Helm"},
			BaseSockets:   2,
		},
		{
			Name:          "Stealth",
			Inserts:       []string{"TalRune", "EthRune"},
			BaseItemTypes: []string{"Armor"},
			BaseSockets:   2,
		},
		{
			Name:          "Lore",
			Inserts:       []string{"OrtRune", "SolRune"},
			BaseItemTypes: []string{"Helm"},
			BaseSockets:   2,
		},
		{
			Name:          "Ancients Pledge",
			Inserts:       []string{"RalRune", "OrtRune", "TalRune"},
			BaseItemTypes: []string{"Shield"},
			BaseSockets:   3,
		},
		{
			Name:          "Smoke",
			Inserts:       []string{"NefRune", "LumRune"},
			BaseItemTypes: []string{"Armor"},
			BaseSockets:   2,
		},
		{
			Name:          "Spirit sword",
			Inserts:       []string{"TalRune", "ThulRune", "OrtRune", "AmnRune"},
			BaseItemTypes: []string{"Sword"},
			BaseSockets:   4,
		},
		{
			Name:          "Spirit shield",
			Inserts:       []string{"TalRune", "ThulRune", "OrtRune", "AmnRune"},
			BaseItemTypes: []string{"Shield", "Auric Shields"},
			BaseSockets:   4,
		},
		{
			Name:          "Insight",
			Inserts:       []string{"RalRune", "TirRune", "TalRune", "SolRune"},
			BaseItemTypes: []string{"Polearm"},
			BaseSockets:   4,
		},
		{
			Name:          "Leaf",
			Inserts:       []string{"TirRune", "RalRune"},
			BaseItemTypes: []string{"Staff"},
			BaseSockets:   2,
		},
	}
)

func alreadyFilled(item data.Item) bool {

	// List of things that can appear on white items
	allowedStats := []stat.ID{
		stat.Defense,
		stat.MinDamage,
		stat.TwoHandedMinDamage,
		stat.MaxDamage,
		stat.TwoHandedMaxDamage,
		stat.AttackRate,
		stat.AttackRating,
		stat.EnhancedDamage,
		stat.EnhancedDamageMax,
		stat.Durability,
		stat.EnhancedDefense,
		stat.MaxDurabilityPercent,
		stat.MaxDurability,
		stat.ChanceToBlock,
		stat.FasterBlockRate,
		stat.NumSockets,
		stat.AddClassSkills,
		stat.NonClassSkill,
		stat.AddSkillTab,
		stat.AllSkills,
	}

	for _, itemStat := range item.Stats {

		// White pala shields can have all resist
		if (itemStat.ID == stat.ColdResist || itemStat.ID == stat.FireResist || itemStat.ID == stat.LightningResist || itemStat.ID == stat.PoisonResist) && item.Type().Name == "Auric Shields" {
			continue
		}

		statAllowed := false
		for _, allowed := range allowedStats {
			if itemStat.ID == allowed {
				statAllowed = true
				break
			}
		}

		if !statAllowed {
			return true // item sockets probably already filled
		}
	}

	return false
}

func hasBaseForSocketRecipe(items []data.Item, sockrecipe SocketRecipe) (data.Item, bool) {

	for _, item := range items {
		itemType := item.Type().Name

		// Check if item type matches any of the allowed base types
		// TODO Allow for multiple bases in inventory and select the best one
		isValidType := false
		for _, baseType := range sockrecipe.BaseItemTypes {
			if itemType == baseType {
				isValidType = true
				break
			}
		}
		if !isValidType {
			continue
		}

		sockets, found := item.FindStat(stat.NumSockets, 0)
		if !found || sockets.Value != sockrecipe.BaseSockets {
			continue
		}

		// Check if item has unwanted stats (already socketed/modified)
		if alreadyFilled(item) {
			continue
		}
		return item, true
	}

	return data.Item{}, false
}

func hasItemsForSocketRecipe(items []data.Item, sockrecipe SocketRecipe) ([]data.Item, bool) {

	socketrecipeItems := make(map[string]int)
	for _, item := range sockrecipe.Inserts {
		socketrecipeItems[item]++
	}

	itemsForRecipe := []data.Item{}

	// Iterate over the items in our stash to see if we have the items for the recipie.
	for _, item := range items {
		if count, ok := socketrecipeItems[string(item.Name)]; ok {

			itemsForRecipe = append(itemsForRecipe, item)

			// Check if we now have exactly the needed count before decrementing
			count -= 1
			if count == 0 {
				delete(socketrecipeItems, string(item.Name))
				if len(socketrecipeItems) == 0 {
					return itemsForRecipe, true
				}
			} else {
				socketrecipeItems[string(item.Name)] = count
			}
		}
	}

	return nil, false
}
func SetSocketRecipes() error {
	ctx := context.Get()
	ctx.SetLastAction("SocketAddItems")

	insertsInStash := ctx.Data.Inventory.ByLocation(item.LocationStash, item.LocationSharedStash)
	basesInStash := ctx.Data.Inventory.ByLocation(item.LocationStash, item.LocationSharedStash)

	for _, recipe := range SocketRecipes {
		if !slices.Contains(ctx.CharacterCfg.SocketRecipes.EnabledSocketRecipes, recipe.Name) {
			continue
		}

		ctx.Logger.Debug("Socket recipe is enabled, processing", "recipe", recipe.Name)

		continueProcessing := true
		for continueProcessing {
			for _, recipe := range SocketRecipes {

				if baseItems, hasBase := hasBaseForSocketRecipe(basesInStash, recipe); hasBase {
					if inserts, hasInserts := hasItemsForSocketRecipe(insertsInStash, recipe); hasInserts {

						TakeItemsFromStash([]data.Item{baseItems})
						baseToUse, _ := ctx.Data.Inventory.FindByID(baseItems.UnitID)
						TakeItemsFromStash(inserts)
						itemsToUse := inserts
						err := SocketItems(ctx, recipe, baseToUse, itemsToUse...)
						if err != nil {
							return err
						}

						stashInventory(true)
						//insertsInStash = RemoveUsedItems(insertsInStash, inserts)
					} else {
						continueProcessing = false
					}
					stashInventory(true)
					//basesInStash = RemoveUsedItems(basesInStash, []data.Item{baseItems})
				} else {
					continueProcessing = false
				}
			}
		}
	}
	return nil
}

func SocketItems(ctx *context.Status, recipe SocketRecipe, base data.Item, items ...data.Item) error {

	ctx.SetLastAction("SocketItem")
	itemsInInv := ctx.Data.Inventory.ByLocation(item.LocationInventory)

	requiredCounts := make(map[string]int)
	for _, insert := range recipe.Inserts {
		requiredCounts[insert]++
	}

	usedItems := make(map[*data.Item]bool)
	orderedItems := make([]data.Item, 0)

	// Process each required insert in order
	for _, requiredInsert := range recipe.Inserts {
		for i := range itemsInInv {
			item := &itemsInInv[i]
			if string(item.Name) == requiredInsert && !usedItems[item] {
				orderedItems = append(orderedItems, *item)
				usedItems[item] = true
				break
			}
		}
	}

	for _, itm := range orderedItems {

		basescreenPos := ui.GetScreenCoordsForItem(base)
		screenPos := ui.GetScreenCoordsForItem(itm)
		ctx.HID.Click(game.LeftButton, screenPos.X, screenPos.Y)
		utils.Sleep(300)

		ctx.HID.Click(game.LeftButton, basescreenPos.X, basescreenPos.Y)
	}

	utils.Sleep(300)

	return step.CloseAllMenus()
}