package keeper_test

import (
	abci "github.com/cometbft/cometbft/abci/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Canto-Network/Canto/v8/app"
	"github.com/Canto-Network/Canto/v8/contracts"
	"github.com/Canto-Network/Canto/v8/x/erc20/types"
	onboardingtest "github.com/Canto-Network/Canto/v8/x/onboarding/testutil"
)

var _ = Describe("Onboarding: Performing an IBC Transfer followed by autoswap and convert", Ordered, func() {
	coincanto := sdk.NewCoin("aepix", sdkmath.ZeroInt())
	ibcBalance := sdk.NewCoin(uusdcIbcdenom, sdkmath.NewIntWithDecimal(10000, 6))
	coinUsdc := sdk.NewCoin("uUSDC", sdkmath.NewIntWithDecimal(10000, 6))
	coinAtom := sdk.NewCoin("uatom", sdkmath.NewIntWithDecimal(10000, 6))

	var (
		sender, receiver string
		senderAcc        sdk.AccAddress
		receiverAcc      sdk.AccAddress
		result           *abci.ExecTxResult
		tokenPair        *types.TokenPair
	)

	BeforeEach(func() {
		s.SetupTest()
	})

	Describe("from a non-authorized channel: Cosmos ---(uatom)---> Canto", func() {
		BeforeEach(func() {
			// deploy ERC20 contract and register token pair
			tokenPair = s.setupRegisterCoin(metadataIbcUSDC)

			// send coins from Cosmos to canto
			sender = s.IBCCosmosChain.SenderAccount.GetAddress().String()
			receiver = s.cantoChain.SenderAccount.GetAddress().String()
			senderAcc = sdk.MustAccAddressFromBech32(sender)
			receiverAcc = sdk.MustAccAddressFromBech32(receiver)
			result = s.SendAndReceiveMessage(s.pathCosmoscanto, s.IBCCosmosChain, "uatom", 10000000000, sender, receiver, 1)

		})
		It("No swap and convert operation - aepix balance should be 0", func() {
			nativecanto := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, "aepix")
			Expect(nativecanto).To(Equal(coincanto))
		})
		It("Canto chain's IBC voucher balance should be same with the transferred amount", func() {
			ibcAtom := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, uatomIbcdenom)
			Expect(ibcAtom).To(Equal(sdk.NewCoin(uatomIbcdenom, coinAtom.Amount)))
		})
		It("Cosmos chain's uatom balance should be 0", func() {
			atom := s.IBCCosmosChain.GetSimApp().BankKeeper.GetBalance(s.IBCCosmosChain.GetContext(), senderAcc, "uatom")
			Expect(atom).To(Equal(sdk.NewCoin("uatom", sdkmath.ZeroInt())))
		})
	})

	Describe("from an authorized channel: Gravity ---(uUSDC)---> Canto", func() {
		When("ERC20 contract is deployed and token pair is enabled", func() {
			BeforeEach(func() {
				// deploy ERC20 contract and register token pair
				tokenPair = s.setupRegisterCoin(metadataIbcUSDC)

				sender = s.IBCGravityChain.SenderAccount.GetAddress().String()
				receiver = s.cantoChain.SenderAccount.GetAddress().String()
				senderAcc = sdk.MustAccAddressFromBech32(sender)
				receiverAcc = sdk.MustAccAddressFromBech32(receiver)

				s.FundCantoChain(sdk.NewCoins(ibcBalance))

			})

			Context("when no swap pool exists", func() {
				BeforeEach(func() {
					result = s.SendAndReceiveMessage(s.pathGravitycanto, s.IBCGravityChain, "uUSDC", 10000000000, sender, receiver, 1)
				})
				It("No swap: aepix balance should be 0", func() {
					nativecanto := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, "aepix")
					Expect(nativecanto).To(Equal(coincanto))
				})
				It("Convert: Canto chain's IBC voucher balance should be same with the original balance", func() {
					ibcUsdc := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, uusdcIbcdenom)
					Expect(ibcUsdc).To(Equal(ibcBalance))
				})
				It("Convert: ERC20 token balance should be same with the transferred IBC voucher amount", func() {
					erc20balance := s.cantoChain.App.(*app.Canto).Erc20Keeper.BalanceOf(s.cantoChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, tokenPair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
					Expect(erc20balance).To(Equal(coinUsdc.Amount.BigInt()))
				})
				It("Convert: ERC20 token balance should be same with the converted IBC voucher amount", func() {
					events := result.GetEvents()
					var sdkEvents []sdk.Event
					for _, event := range events {
						sdkEvents = append(sdkEvents, sdk.Event(event))
					}

					attrs := onboardingtest.ExtractAttributes(onboardingtest.FindEvent(sdkEvents, "convert_coin"))
					convertAmount, _ := sdkmath.NewIntFromString(attrs["amount"])
					erc20balance := s.cantoChain.App.(*app.Canto).Erc20Keeper.BalanceOf(s.cantoChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, tokenPair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
					Expect(erc20balance).To(Equal(convertAmount.BigInt()))
				})
			})

			Context("when swap pool exists", func() {
				BeforeEach(func() {
					s.CreatePool(uusdcIbcdenom)
				})
				When("aepix balance is 0 and not enough IBC token transferred to swap aepix", func() {
					BeforeEach(func() {
						result = s.SendAndReceiveMessage(s.pathGravitycanto, s.IBCGravityChain, "uUSDC", 1000000, sender, receiver, 1)
					})
					It("No swap: Balance of aepix should be same with the original aepix balance (0)", func() {
						nativecanto := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, "aepix")
						Expect(nativecanto).To(Equal(sdk.NewCoin("aepix", sdkmath.ZeroInt())))
					})
					It("Convert: Canto chain's IBC voucher balance should be same with the original balance", func() {
						ibcUsdc := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, uusdcIbcdenom)
						Expect(ibcUsdc).To(Equal(ibcBalance))
					})
					It("Convert: ERC20 token balance should be same with the transferred IBC voucher amount", func() {
						erc20balance := s.cantoChain.App.(*app.Canto).Erc20Keeper.BalanceOf(s.cantoChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, tokenPair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
						Expect(erc20balance).To(Equal(sdkmath.NewIntWithDecimal(1, 6).BigInt()))
					})
					It("Convert: ERC20 token balance should be same with the converted IBC voucher amount", func() {
						events := result.GetEvents()
						var sdkEvents []sdk.Event
						for _, event := range events {
							sdkEvents = append(sdkEvents, sdk.Event(event))
						}
						attrs := onboardingtest.ExtractAttributes(onboardingtest.FindEvent(sdkEvents, "convert_coin"))
						convertAmount, _ := sdkmath.NewIntFromString(attrs["amount"])
						erc20balance := s.cantoChain.App.(*app.Canto).Erc20Keeper.BalanceOf(s.cantoChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, tokenPair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
						Expect(erc20balance).To(Equal(convertAmount.BigInt()))
					})
				})

				When("Canto chain's aepix balance is 0", func() {
					BeforeEach(func() {
						result = s.SendAndReceiveMessage(s.pathGravitycanto, s.IBCGravityChain, "uUSDC", 10000000000, sender, receiver, 1)
					})
					It("Swap: balance of aepix should be same with the auto swap threshold", func() {
						autoSwapThreshold := s.cantoChain.App.(*app.Canto).OnboardingKeeper.GetParams(s.cantoChain.GetContext()).AutoSwapThreshold
						nativecanto := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, "aepix")
						Expect(nativecanto).To(Equal(sdk.NewCoin("aepix", autoSwapThreshold)))
					})
					It("Convert: Canto chain's IBC voucher balance should be same with the original balance", func() {
						ibcUsdc := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, uusdcIbcdenom)
						Expect(ibcUsdc).To(Equal(ibcBalance))
					})
					It("Convert: ERC20 token balance should be same with the difference between transferred IBC voucher amount and the swapped amount", func() {
						events := result.GetEvents()
						var sdkEvents []sdk.Event
						for _, event := range events {
							sdkEvents = append(sdkEvents, sdk.Event(event))
						}
						attrs := onboardingtest.ExtractAttributes(onboardingtest.FindEvent(sdkEvents, "swap"))
						swappedAmount, _ := sdkmath.NewIntFromString(attrs["amount"])
						erc20balance := s.cantoChain.App.(*app.Canto).Erc20Keeper.BalanceOf(s.cantoChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, tokenPair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
						Expect(erc20balance).To(Equal(coinUsdc.Amount.Sub(swappedAmount).BigInt()))
					})
					It("Convert: ERC20 token balance should be same with the converted IBC voucher amount", func() {
						events := result.GetEvents()
						var sdkEvents []sdk.Event
						for _, event := range events {
							sdkEvents = append(sdkEvents, sdk.Event(event))
						}
						attrs := onboardingtest.ExtractAttributes(onboardingtest.FindEvent(sdkEvents, "convert_coin"))
						convertAmount, _ := sdkmath.NewIntFromString(attrs["amount"])
						erc20balance := s.cantoChain.App.(*app.Canto).Erc20Keeper.BalanceOf(s.cantoChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, tokenPair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
						Expect(erc20balance).To(Equal(convertAmount.BigInt()))
					})
				})

				When("Canto chain's aepix balance is between 0 and auto swap threshold (3canto)", func() {
					BeforeEach(func() {
						s.FundCantoChain(sdk.NewCoins(sdk.NewCoin("aepix", sdkmath.NewIntWithDecimal(3, 18))))
						result = s.SendAndReceiveMessage(s.pathGravitycanto, s.IBCGravityChain, "uUSDC", 10000000000, sender, receiver, 1)
					})
					It("Auto swap operation: balance of aepix should be same with the sum of original aepix balance and auto swap threshold", func() {
						autoSwapThreshold := s.cantoChain.App.(*app.Canto).OnboardingKeeper.GetParams(s.cantoChain.GetContext()).AutoSwapThreshold
						nativecanto := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, "aepix")
						Expect(nativecanto).To(Equal(sdk.NewCoin("aepix", autoSwapThreshold.Add(sdkmath.NewIntWithDecimal(3, 18)))))
					})
					It("Convert: Canto chain's IBC voucher balance should be same with the original balance", func() {
						ibcUsdc := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, uusdcIbcdenom)
						Expect(ibcUsdc).To(Equal(ibcBalance))
					})
					It("Convert: ERC20 token balance should be same with the difference between transferred IBC voucher amount and the swapped amount", func() {
						events := result.GetEvents()
						var sdkEvents []sdk.Event
						for _, event := range events {
							sdkEvents = append(sdkEvents, sdk.Event(event))
						}
						attrs := onboardingtest.ExtractAttributes(onboardingtest.FindEvent(sdkEvents, "swap"))
						swappedAmount, _ := sdkmath.NewIntFromString(attrs["amount"])
						erc20balance := s.cantoChain.App.(*app.Canto).Erc20Keeper.BalanceOf(s.cantoChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, tokenPair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
						Expect(erc20balance).To(Equal(coinUsdc.Amount.Sub(swappedAmount).BigInt()))
					})
					It("Convert: ERC20 token balance should be same with the converted IBC voucher amount", func() {
						events := result.GetEvents()
						var sdkEvents []sdk.Event
						for _, event := range events {
							sdkEvents = append(sdkEvents, sdk.Event(event))
						}
						attrs := onboardingtest.ExtractAttributes(onboardingtest.FindEvent(sdkEvents, "convert_coin"))
						convertAmount, _ := sdkmath.NewIntFromString(attrs["amount"])
						erc20balance := s.cantoChain.App.(*app.Canto).Erc20Keeper.BalanceOf(s.cantoChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, tokenPair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
						Expect(erc20balance).To(Equal(convertAmount.BigInt()))
					})
				})
				When("Canto chain's aepix balance is bigger than the auto swap threshold (4canto)", func() {
					BeforeEach(func() {
						s.FundCantoChain(sdk.NewCoins(sdk.NewCoin("aepix", sdkmath.NewIntWithDecimal(4, 18))))
						result = s.SendAndReceiveMessage(s.pathGravitycanto, s.IBCGravityChain, "uUSDC", 10000000000, sender, receiver, 1)
					})
					It("No swap: balance of aepix should be same with the original aepix balance (4canto)", func() {
						nativecanto := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, "aepix")
						Expect(nativecanto).To(Equal(sdk.NewCoin("aepix", sdkmath.NewIntWithDecimal(4, 18))))
					})
					It("Convert: Canto chain's IBC voucher balance should be same with the original balance", func() {
						ibcUsdc := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, uusdcIbcdenom)
						Expect(ibcUsdc).To(Equal(ibcBalance))
					})
					It("Convert: ERC20 token balance should be same with the transferred IBC voucher amount", func() {
						erc20balance := s.cantoChain.App.(*app.Canto).Erc20Keeper.BalanceOf(s.cantoChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, tokenPair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
						Expect(erc20balance).To(Equal(coinUsdc.Amount.BigInt()))
					})
					It("Convert: ERC20 token balance should be same with the converted IBC voucher amount", func() {
						events := result.GetEvents()
						var sdkEvents []sdk.Event
						for _, event := range events {
							sdkEvents = append(sdkEvents, sdk.Event(event))
						}
						attrs := onboardingtest.ExtractAttributes(onboardingtest.FindEvent(sdkEvents, "convert_coin"))
						convertAmount, _ := sdkmath.NewIntFromString(attrs["amount"])
						erc20balance := s.cantoChain.App.(*app.Canto).Erc20Keeper.BalanceOf(s.cantoChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, tokenPair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
						Expect(erc20balance).To(Equal(convertAmount.BigInt()))
					})
				})
			})
		})
		When("ERC20 contract is deployed and token pair is enabled", func() {
			BeforeEach(func() {
				// deploy ERC20 contract and register token pair
				tokenPair = s.setupRegisterCoin(metadataIbcUSDC)
				tokenPair.Enabled = false
				s.cantoChain.App.(*app.Canto).Erc20Keeper.SetTokenPair(s.cantoChain.GetContext(), *tokenPair)
				sender = s.IBCGravityChain.SenderAccount.GetAddress().String()
				receiver = s.cantoChain.SenderAccount.GetAddress().String()
				senderAcc = sdk.MustAccAddressFromBech32(sender)
				receiverAcc = sdk.MustAccAddressFromBech32(receiver)

				s.CreatePool(uusdcIbcdenom)
				s.FundCantoChain(sdk.NewCoins(ibcBalance))
				s.FundCantoChain(sdk.NewCoins(sdk.NewCoin("aepix", sdkmath.NewIntWithDecimal(3, 18))))
				result = s.SendAndReceiveMessage(s.pathGravitycanto, s.IBCGravityChain, "uUSDC", 10000000000, sender, receiver, 1)

			})
			It("Auto swap operation: balance of aepix should be same with the sum of original aepix balance and auto swap threshold", func() {
				autoSwapThreshold := s.cantoChain.App.(*app.Canto).OnboardingKeeper.GetParams(s.cantoChain.GetContext()).AutoSwapThreshold
				nativecanto := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, "aepix")
				Expect(nativecanto).To(Equal(sdk.NewCoin("aepix", autoSwapThreshold.Add(sdkmath.NewIntWithDecimal(3, 18)))))
			})
			It("No convert: Canto chain's IBC voucher balance should be same with (original balance + transferred amount - swapped amount)", func() {
				events := result.GetEvents()
				var sdkEvents []sdk.Event
				for _, event := range events {
					sdkEvents = append(sdkEvents, sdk.Event(event))
				}
				attrs := onboardingtest.ExtractAttributes(onboardingtest.FindEvent(sdkEvents, "swap"))
				swappedAmount, _ := sdkmath.NewIntFromString(attrs["amount"])
				ibcUsdc := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, uusdcIbcdenom)
				Expect(ibcUsdc.Amount).To(Equal(ibcBalance.Amount.Add(sdkmath.NewInt(10000000000)).Sub(swappedAmount)))
			})
		})
		When("ERC20 contract is not deployed", func() {
			BeforeEach(func() {
				sender = s.IBCGravityChain.SenderAccount.GetAddress().String()
				receiver = s.cantoChain.SenderAccount.GetAddress().String()
				senderAcc = sdk.MustAccAddressFromBech32(sender)
				receiverAcc = sdk.MustAccAddressFromBech32(receiver)

				s.CreatePool(uusdcIbcdenom)
				s.FundCantoChain(sdk.NewCoins(ibcBalance))
				s.FundCantoChain(sdk.NewCoins(sdk.NewCoin("aepix", sdkmath.NewIntWithDecimal(3, 18))))
				result = s.SendAndReceiveMessage(s.pathGravitycanto, s.IBCGravityChain, "uUSDC", 10000000000, sender, receiver, 1)
			})
			It("Auto swap operation: balance of aepix should be same with the sum of original aepix balance and auto swap threshold", func() {
				autoSwapThreshold := s.cantoChain.App.(*app.Canto).OnboardingKeeper.GetParams(s.cantoChain.GetContext()).AutoSwapThreshold
				nativecanto := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, "aepix")
				Expect(nativecanto).To(Equal(sdk.NewCoin("aepix", autoSwapThreshold.Add(sdkmath.NewIntWithDecimal(3, 18)))))
			})
			It("No convert: Canto chain's IBC voucher balance should be same with (original balance + transferred amount - swapped amount)", func() {
				events := result.GetEvents()
				var sdkEvents []sdk.Event
				for _, event := range events {
					sdkEvents = append(sdkEvents, sdk.Event(event))
				}
				attrs := onboardingtest.ExtractAttributes(onboardingtest.FindEvent(sdkEvents, "swap"))
				swappedAmount, _ := sdkmath.NewIntFromString(attrs["amount"])
				ibcUsdc := s.cantoChain.App.(*app.Canto).BankKeeper.GetBalance(s.cantoChain.GetContext(), receiverAcc, uusdcIbcdenom)
				Expect(ibcUsdc.Amount).To(Equal(ibcBalance.Amount.Add(sdkmath.NewInt(10000000000)).Sub(swappedAmount)))
			})

		})
	})

})
